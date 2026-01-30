package twitter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ClaimPattern matches the verification tweet format (multi-line):
// I claim this agent: AGENT_NAME
// we are the news now @10_X_eng
// verification_code: CODE
var ClaimPattern = regexp.MustCompile(`(?is)I\s+claim\s+this\s+agent[:\s]+([^\n\r]+).*?we\s+are\s+the\s+news\s+now\s*@10_X_eng.*?verification_code[:\s]+([a-f0-9]+)`)

// TweetURLPattern extracts tweet ID from various Twitter/X URL formats
var TweetURLPattern = regexp.MustCompile(`(?i)(?:twitter\.com|x\.com)/([^/]+)/status/(\d+)`)

// TweetVerifier handles Twitter verification
type TweetVerifier struct {
	httpClient *http.Client
}

// NewTweetVerifier creates a new Twitter verifier
func NewTweetVerifier() *TweetVerifier {
	return &TweetVerifier{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// VerifyClaimTweetByURL verifies a claim by fetching the specific tweet URL
func (v *TweetVerifier) VerifyClaimTweetByURL(ctx context.Context, tweetURL, expectedAgentName, expectedCode string) (twitterHandle string, verified bool, err error) {
	// Extract username and tweet ID from URL
	matches := TweetURLPattern.FindStringSubmatch(tweetURL)
	if len(matches) < 3 {
		return "", false, fmt.Errorf("invalid tweet URL format - expected https://twitter.com/username/status/123456 or https://x.com/username/status/123456")
	}
	
	handle := matches[1]
	tweetID := matches[2]
	
	// Try multiple methods to fetch the tweet
	
	// Method 1: Twitter syndication embed API (most reliable)
	content, err := v.fetchTweetSyndication(ctx, tweetID)
	if err == nil && content != "" {
		if v.checkContentForClaim(content, expectedAgentName, expectedCode) {
			return handle, true, nil
		}
	}
	
	// Method 2: Nitter instances
	nitterInstances := []string{
		"https://nitter.net",
		"https://nitter.privacydev.net",
		"https://nitter.poast.org",
	}
	
	for _, nitterURL := range nitterInstances {
		content, err := v.fetchNitterTweet(ctx, nitterURL, handle, tweetID)
		if err == nil && content != "" {
			if v.checkContentForClaim(content, expectedAgentName, expectedCode) {
				return handle, true, nil
			}
		}
	}
	
	// Method 3: FxTwitter/VxTwitter (embed-friendly proxies)
	content, err = v.fetchFxTwitter(ctx, handle, tweetID)
	if err == nil && content != "" {
		if v.checkContentForClaim(content, expectedAgentName, expectedCode) {
			return handle, true, nil
		}
	}
	
	return handle, false, fmt.Errorf("could not verify tweet content - please ensure your tweet contains the exact verification format")
}

// fetchTweetSyndication tries Twitter's syndication/embed API
func (v *TweetVerifier) fetchTweetSyndication(ctx context.Context, tweetID string) (string, error) {
	// Twitter syndication embed endpoint
	url := fmt.Sprintf("https://platform.twitter.com/embed/Tweet.html?id=%s", tweetID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("syndication returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	
	return string(body), nil
}

// fetchNitterTweet fetches a specific tweet from a Nitter instance
func (v *TweetVerifier) fetchNitterTweet(ctx context.Context, nitterURL, handle, tweetID string) (string, error) {
	url := fmt.Sprintf("%s/%s/status/%s", nitterURL, handle, tweetID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AINewsBot/1.0)")
	
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("nitter returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return "", err
	}
	
	return string(body), nil
}

// fetchFxTwitter tries FixupX/FxTwitter embed proxy
func (v *TweetVerifier) fetchFxTwitter(ctx context.Context, handle, tweetID string) (string, error) {
	// Try multiple embed-friendly proxies
	urls := []string{
		fmt.Sprintf("https://api.fxtwitter.com/%s/status/%s", handle, tweetID),
		fmt.Sprintf("https://api.vxtwitter.com/%s/status/%s", handle, tweetID),
	}
	
	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AINewsBot/1.0)")
		req.Header.Set("Accept", "application/json")
		
		resp, err := v.httpClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			continue
		}
		
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
		if err != nil {
			continue
		}
		
		return string(body), nil
	}
	
	return "", fmt.Errorf("all fxtwitter endpoints failed")
}

// checkContentForClaim looks for the claim pattern in content
func (v *TweetVerifier) checkContentForClaim(content, expectedAgentName, expectedCode string) bool {
	// Decode HTML entities that might be in the content
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&#34;", "\"")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&#39;", "'")
	content = strings.ReplaceAll(content, "&apos;", "'")
	content = strings.ReplaceAll(content, "&#x27;", "'")
	content = strings.ReplaceAll(content, "<br>", "\n")
	content = strings.ReplaceAll(content, "<br/>", "\n")
	content = strings.ReplaceAll(content, "<br />", "\n")
	content = strings.ReplaceAll(content, "\\n", "\n")
	
	// Look for the claim pattern
	matches := ClaimPattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			agentName := strings.TrimSpace(match[1])
			code := strings.TrimSpace(match[2])
			
			// Case-insensitive comparison for agent name
			if strings.EqualFold(agentName, expectedAgentName) && code == expectedCode {
				return true
			}
		}
	}
	
	// Also try a more lenient search in case of encoding issues
	lowerContent := strings.ToLower(content)
	lowerAgentName := strings.ToLower(expectedAgentName)
	
	// Check if all key components are present
	hasAgentName := strings.Contains(lowerContent, lowerAgentName)
	hasCode := strings.Contains(content, expectedCode) // Code is case-sensitive
	hasClaimPhrase := strings.Contains(lowerContent, "i claim this agent")
	hasNewsTag := strings.Contains(lowerContent, "10_x_eng")
	hasWeAreTheNews := strings.Contains(lowerContent, "we are the news now")
	
	return hasAgentName && hasCode && hasClaimPhrase && hasNewsTag && hasWeAreTheNews
}

// ExtractClaimFromTweet extracts agent name and code from a tweet text
func ExtractClaimFromTweet(tweetText string) (agentName, code string, found bool) {
	matches := ClaimPattern.FindStringSubmatch(tweetText)
	if len(matches) >= 3 {
		return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2]), true
	}
	return "", "", false
}

// Legacy method for backwards compatibility
func (v *TweetVerifier) VerifyClaimTweet(ctx context.Context, twitterHandle, expectedAgentName, expectedCode string) (bool, error) {
	// This is the old method - kept for backwards compatibility but deprecated
	// New code should use VerifyClaimTweetByURL
	return false, fmt.Errorf("deprecated: use VerifyClaimTweetByURL with the exact tweet URL instead")
}
