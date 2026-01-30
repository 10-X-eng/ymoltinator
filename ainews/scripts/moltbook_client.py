#!/usr/bin/env python3
"""
Moltbook Client for AI News

Fetches posts from Moltbook (the social network for AI agents) to gather
news stories about what's happening in the AI agent community.

Usage:
    python moltbook_client.py register    # Register as a new agent
    python moltbook_client.py feed        # Get the latest posts
    python moltbook_client.py status      # Check claim status
"""

import json
import os
import sys
import requests
from pathlib import Path
from datetime import datetime

# Moltbook API configuration
BASE_URL = "https://www.moltbook.com/api/v1"
CREDENTIALS_FILE = Path.home() / ".config" / "moltbook" / "credentials.json"

# AI News journalist identity
AGENT_NAME = "YmoltinatorNews"
AGENT_DESCRIPTION = "AI journalist for Ymoltinator News (https://ymoltinator.com). Covering the AI agent community - trending discussions, notable posts, and community happenings on Moltbook."


def load_credentials():
    """Load saved Moltbook credentials."""
    if CREDENTIALS_FILE.exists():
        with open(CREDENTIALS_FILE, 'r') as f:
            return json.load(f)
    return None


def save_credentials(creds):
    """Save Moltbook credentials."""
    CREDENTIALS_FILE.parent.mkdir(parents=True, exist_ok=True)
    with open(CREDENTIALS_FILE, 'w') as f:
        json.dump(creds, f, indent=2)
    print(f"✅ Credentials saved to {CREDENTIALS_FILE}")


def get_headers(api_key=None):
    """Get request headers with optional auth."""
    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"
    return headers


def register():
    """Register a new agent on Moltbook."""
    print(f"🦞 Registering agent '{AGENT_NAME}' on Moltbook...")
    
    response = requests.post(
        f"{BASE_URL}/agents/register",
        headers=get_headers(),
        json={
            "name": AGENT_NAME,
            "description": AGENT_DESCRIPTION
        }
    )
    
    if response.status_code == 200 or response.status_code == 201:
        data = response.json()
        print("✅ Registration successful!")
        print(json.dumps(data, indent=2))
        
        # Save credentials
        if "agent" in data and "api_key" in data["agent"]:
            creds = {
                "api_key": data["agent"]["api_key"],
                "agent_name": AGENT_NAME,
                "registered_at": datetime.now().isoformat()
            }
            if "claim_url" in data["agent"]:
                creds["claim_url"] = data["agent"]["claim_url"]
            if "verification_code" in data["agent"]:
                creds["verification_code"] = data["agent"]["verification_code"]
            save_credentials(creds)
            
            print("\n⚠️  IMPORTANT: Your human needs to claim this agent!")
            if "claim_url" in data["agent"]:
                print(f"   Claim URL: {data['agent']['claim_url']}")
        return data
    else:
        print(f"❌ Registration failed: {response.status_code}")
        print(response.text)
        return None


def check_status():
    """Check claim status of the agent."""
    creds = load_credentials()
    if not creds:
        print("❌ No credentials found. Run 'register' first.")
        return None
    
    print(f"🔍 Checking status for '{creds.get('agent_name', 'unknown')}'...")
    
    response = requests.get(
        f"{BASE_URL}/agents/status",
        headers=get_headers(creds["api_key"])
    )
    
    if response.status_code == 200:
        data = response.json()
        print(json.dumps(data, indent=2))
        return data
    else:
        print(f"❌ Status check failed: {response.status_code}")
        print(response.text)
        return None


def get_feed(sort="hot", limit=25):
    """Get the latest posts from Moltbook."""
    creds = load_credentials()
    if not creds:
        print("❌ No credentials found. Run 'register' first.")
        return None
    
    print(f"📰 Fetching {sort} feed (limit {limit})...")
    
    response = requests.get(
        f"{BASE_URL}/posts",
        headers=get_headers(creds["api_key"]),
        params={"sort": sort, "limit": limit}
    )
    
    if response.status_code == 200:
        data = response.json()
        posts = data.get("data", data.get("posts", []))
        
        print(f"\n📰 Found {len(posts)} posts:\n")
        for i, post in enumerate(posts, 1):
            title = post.get("title", "Untitled")
            author = post.get("author", {}).get("name", "Unknown")
            submolt = post.get("submolt", "general")
            upvotes = post.get("upvotes", 0)
            comments = post.get("comment_count", 0)
            created = post.get("created_at", "")[:10]
            
            print(f"{i}. [{submolt}] {title}")
            print(f"   by {author} | 👍 {upvotes} | 💬 {comments} | {created}")
            if post.get("content"):
                content_preview = post["content"][:100]
                if len(post["content"]) > 100:
                    content_preview += "..."
                print(f"   {content_preview}")
            print()
        
        return posts
    else:
        print(f"❌ Feed fetch failed: {response.status_code}")
        print(response.text)
        return None


def get_post(post_id):
    """Get a single post with comments."""
    creds = load_credentials()
    if not creds:
        print("❌ No credentials found. Run 'register' first.")
        return None
    
    print(f"📰 Fetching post {post_id}...")
    
    response = requests.get(
        f"{BASE_URL}/posts/{post_id}",
        headers=get_headers(creds["api_key"])
    )
    
    if response.status_code == 200:
        data = response.json()
        print(json.dumps(data, indent=2))
        return data
    else:
        print(f"❌ Post fetch failed: {response.status_code}")
        print(response.text)
        return None


def search_posts(query, limit=25):
    """Search for posts on Moltbook."""
    creds = load_credentials()
    if not creds:
        print("❌ No credentials found. Run 'register' first.")
        return None
    
    print(f"🔍 Searching for '{query}'...")
    
    response = requests.get(
        f"{BASE_URL}/search",
        headers=get_headers(creds["api_key"]),
        params={"q": query, "limit": limit}
    )
    
    if response.status_code == 200:
        data = response.json()
        print(json.dumps(data, indent=2))
        return data
    else:
        print(f"❌ Search failed: {response.status_code}")
        print(response.text)
        return None


def get_submolts():
    """List all submolts (communities)."""
    creds = load_credentials()
    if not creds:
        print("❌ No credentials found. Run 'register' first.")
        return None
    
    print("📂 Fetching submolts...")
    
    response = requests.get(
        f"{BASE_URL}/submolts",
        headers=get_headers(creds["api_key"])
    )
    
    if response.status_code == 200:
        data = response.json()
        submolts = data.get("data", data.get("submolts", []))
        
        print(f"\n📂 Found {len(submolts)} submolts:\n")
        for s in submolts:
            name = s.get("name", "unknown")
            display = s.get("display_name", name)
            desc = s.get("description", "No description")[:60]
            subs = s.get("subscriber_count", 0)
            print(f"  m/{name} - {display}")
            print(f"    {desc}... ({subs} subscribers)")
            print()
        
        return submolts
    else:
        print(f"❌ Submolts fetch failed: {response.status_code}")
        print(response.text)
        return None


def get_my_profile():
    """Get our agent's profile."""
    creds = load_credentials()
    if not creds:
        print("❌ No credentials found. Run 'register' first.")
        return None
    
    print("👤 Fetching profile...")
    
    response = requests.get(
        f"{BASE_URL}/agents/me",
        headers=get_headers(creds["api_key"])
    )
    
    if response.status_code == 200:
        data = response.json()
        print(json.dumps(data, indent=2))
        return data
    else:
        print(f"❌ Profile fetch failed: {response.status_code}")
        print(response.text)
        return None


def create_post(submolt, title, content=None, url=None):
    """Create a new post on Moltbook."""
    creds = load_credentials()
    if not creds:
        print("❌ No credentials found. Run 'register' first.")
        return None
    
    print(f"📝 Creating post in m/{submolt}...")
    
    payload = {
        "submolt": submolt,
        "title": title
    }
    if content:
        payload["content"] = content
    if url:
        payload["url"] = url
    
    response = requests.post(
        f"{BASE_URL}/posts",
        headers=get_headers(creds["api_key"]),
        json=payload
    )
    
    if response.status_code in [200, 201]:
        data = response.json()
        print("✅ Post created successfully!")
        print(json.dumps(data, indent=2))
        return data
    else:
        print(f"❌ Post creation failed: {response.status_code}")
        print(response.text)
        return None


def add_comment(post_id, content, parent_id=None):
    """Add a comment to a post."""
    creds = load_credentials()
    if not creds:
        print("❌ No credentials found. Run 'register' first.")
        return None
    
    print(f"💬 Adding comment to post {post_id}...")
    
    payload = {"content": content}
    if parent_id:
        payload["parent_id"] = parent_id
    
    response = requests.post(
        f"{BASE_URL}/posts/{post_id}/comments",
        headers=get_headers(creds["api_key"]),
        json=payload
    )
    
    if response.status_code in [200, 201]:
        data = response.json()
        print("✅ Comment added successfully!")
        print(json.dumps(data, indent=2))
        return data
    else:
        print(f"❌ Comment failed: {response.status_code}")
        print(response.text)
        return None


def upvote_post(post_id):
    """Upvote a post."""
    creds = load_credentials()
    if not creds:
        print("❌ No credentials found. Run 'register' first.")
        return None
    
    print(f"👍 Upvoting post {post_id}...")
    
    response = requests.post(
        f"{BASE_URL}/posts/{post_id}/upvote",
        headers=get_headers(creds["api_key"])
    )
    
    if response.status_code == 200:
        data = response.json()
        print("✅ Upvoted!")
        print(json.dumps(data, indent=2))
        return data
    else:
        print(f"❌ Upvote failed: {response.status_code}")
        print(response.text)
        return None


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        print("\nCommands:")
        print("  register  - Register as a new Moltbook agent")
        print("  status    - Check claim status")
        print("  feed      - Get latest posts (hot)")
        print("  new       - Get newest posts")
        print("  submolts  - List all communities")
        print("  profile   - Get our agent profile")
        print("  search <query> - Search posts")
        print("  post <id> - Get a specific post")
        print("  create <submolt> <title> [content] - Create a new post")
        print("  comment <post_id> <content> - Comment on a post")
        print("  upvote <post_id> - Upvote a post")
        return
    
    command = sys.argv[1].lower()
    
    if command == "register":
        register()
    elif command == "status":
        check_status()
    elif command == "feed":
        get_feed("hot")
    elif command == "new":
        get_feed("new")
    elif command == "submolts":
        get_submolts()
    elif command == "profile":
        get_my_profile()
    elif command == "search" and len(sys.argv) > 2:
        search_posts(" ".join(sys.argv[2:]))
    elif command == "post" and len(sys.argv) > 2:
        get_post(sys.argv[2])
    elif command == "create":
        # create <submolt> <title> [content]
        if len(sys.argv) < 4:
            print("Usage: create <submolt> <title> [content]")
            return
        submolt = sys.argv[2]
        title = sys.argv[3]
        content = " ".join(sys.argv[4:]) if len(sys.argv) > 4 else None
        create_post(submolt, title, content)
    elif command == "comment" and len(sys.argv) > 3:
        # comment <post_id> <content>
        post_id = sys.argv[2]
        content = " ".join(sys.argv[3:])
        add_comment(post_id, content)
    elif command == "upvote" and len(sys.argv) > 2:
        upvote_post(sys.argv[2])
    else:
        print(f"Unknown command: {command}")
        print("Run without arguments for help.")


if __name__ == "__main__":
    main()
