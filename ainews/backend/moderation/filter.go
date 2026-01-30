package moderation

import (
	"regexp"
	"strings"
	"sync"
)

// Filter provides content moderation functionality
// This is a LIGHT filter focused on:
// - Actual hate speech and slurs
// - Explicit sexual content (porn sites, graphic descriptions)
// - Direct threats and calls for violence
// It does NOT block:
// - News-worthy words (death, attack, terrorism, etc.)
// - Mild profanity (damn, hell, crap)
// - Drug names in news context
// - Political/cultural terms
type Filter struct {
	badWords    map[string]bool
	patterns    []*regexp.Regexp
	mu          sync.RWMutex
}

// NewFilter creates a new content filter with focused moderation
func NewFilter() *Filter {
	f := &Filter{
		badWords: make(map[string]bool),
	}
	f.loadBadWords()
	f.compilePatterns()
	return f
}

// loadBadWords loads prohibited words - focused on actual harmful content
func (f *Filter) loadBadWords() {
	// FOCUSED list: only truly harmful content that has no legitimate news use
	words := []string{
		// Racial slurs - no legitimate use
		"nigger", "niggers", "nigga", "niggas",
		"chink", "chinks",
		"gook", "gooks",
		"spic", "spics", "spick", "spicks",
		"wetback", "wetbacks",
		"beaner", "beaners",
		"kike", "kikes",
		"coon", "coons",
		"darkie", "darkies",
		"towelhead", "towelheads",
		"raghead", "ragheads",
		"sandnigger", "sandniggers",
		"porch monkey",
		"jungle bunny",
		
		// LGBTQ+ slurs - no legitimate use
		"faggot", "faggots",
		"tranny", "trannies",
		"shemale", "shemales",
		
		// Severe profanity combos
		"cocksucker", "cocksuckers",
		"motherfucker", "motherfuckers", "motherfucking",
		
		// Child exploitation - zero tolerance
		"pedophile", "pedophiles", "pedophilia",
		"child porn",
		
		// Direct self-harm encouragement
		"kill yourself",
		"neck yourself",
		"drink bleach",
		
		// Explicit porn site names (not general words)
		"pornhub",
		"xvideos",
		"xhamster",
		"redtube",
		"youporn",
		"brazzers",
		"bangbros",
		
		// Nazi/hate group specific phrases
		"heil hitler",
		"sieg heil",
		"white power",
		"14 words",
		"blood and soil",
		"jews will not replace us",
		
		// Crypto/financial scam patterns
		"double your bitcoin",
		"nigerian prince",
	}
	
	for _, word := range words {
		f.badWords[strings.ToLower(word)] = true
	}
}

// compilePatterns compiles regex patterns for obfuscated slurs
func (f *Filter) compilePatterns() {
	patterns := []string{
		// Obfuscated racial slurs only
		`n[i1\*@]gg[ae@]r?s?`,
		`f[a@]gg?[o0]t`,
		
		// Porn site URL patterns
		`(?i)(pornhub|xvideos|xhamster|redtube|youporn|brazzers)\.`,
	}
	
	f.patterns = make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if re, err := regexp.Compile(`(?i)` + pattern); err == nil {
			f.patterns = append(f.patterns, re)
		}
	}
}

// ContainsBadWords checks if the text contains any prohibited content
// Returns true if bad content found, along with the matched word/pattern
func (f *Filter) ContainsBadWords(text string) (bool, string) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	lowerText := strings.ToLower(text)
	
	// Check word boundaries for exact matches
	words := tokenize(lowerText)
	for _, word := range words {
		if f.badWords[word] {
			return true, word
		}
	}
	
	// Check for multi-word phrases
	for phrase := range f.badWords {
		if strings.Contains(phrase, " ") && strings.Contains(lowerText, phrase) {
			return true, phrase
		}
	}
	
	// Check regex patterns
	for _, pattern := range f.patterns {
		if match := pattern.FindString(lowerText); match != "" {
			return true, match
		}
	}
	
	return false, ""
}

// tokenize splits text into words, handling various separators
func tokenize(text string) []string {
	// Replace common separators with spaces
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ",
		";", " ", ":", " ", "'", " ", "\"", " ",
		"(", " ", ")", " ", "[", " ", "]", " ",
		"{", " ", "}", " ", "/", " ", "\\", " ",
		"|", " ", "\n", " ", "\r", " ", "\t", " ",
		"-", " ", "_", " ",
	)
	cleaned := replacer.Replace(text)
	
	// Split and filter empty strings
	parts := strings.Fields(cleaned)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) > 0 {
			result = append(result, part)
		}
	}
	return result
}

// ValidateContent validates both title and content of a story
func (f *Filter) ValidateContent(title, content, url string) (bool, string) {
	// Check title
	if found, word := f.ContainsBadWords(title); found {
		return false, "Title contains prohibited content: " + word
	}
	
	// Check content
	if found, word := f.ContainsBadWords(content); found {
		return false, "Content contains prohibited content: " + word
	}
	
	// Check URL
	if found, word := f.ContainsBadWords(url); found {
		return false, "URL contains prohibited content: " + word
	}
	
	return true, ""
}
