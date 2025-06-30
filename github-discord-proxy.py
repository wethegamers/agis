#!/usr/bin/env python3
"""
GitHub to Discord Webhook Proxy
Receives GitHub webhook events and formats them for Discord
"""

import json
import os
from http.server import HTTPServer, BaseHTTPRequestHandler
import urllib.request
import urllib.parse
from datetime import datetime

DISCORD_WEBHOOK_URL = os.environ.get('DISCORD_WEBHOOK_URL', '')
PORT = int(os.environ.get('PORT', '8080'))

class GitHubWebhookHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        try:
            # Get content length and read the payload
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            # Parse GitHub event
            event_type = self.headers.get('X-GitHub-Event', 'unknown')
            payload = json.loads(post_data.decode('utf-8'))
            
            # Handle ping event (GitHub webhook test)
            if event_type == 'ping':
                discord_message = {
                    "embeds": [{
                        "title": "🏓 GitHub Webhook Connected",
                        "description": f"Webhook successfully connected to repository: **{payload.get('repository', {}).get('full_name', 'unknown')}**",
                        "color": 3066993,
                        "timestamp": datetime.utcnow().isoformat() + "Z"
                    }]
                }
            
            # Handle push events
            elif event_type == 'push':
                repo = payload.get('repository', {})
                pusher = payload.get('pusher', {})
                commits = payload.get('commits', [])
                ref = payload.get('ref', '')
                branch = ref.replace('refs/heads/', '') if ref.startswith('refs/heads/') else ref
                
                discord_message = {
                    "embeds": [{
                        "title": f"📤 Push to {repo.get('name', 'unknown')}",
                        "description": f"**{len(commits)}** commit(s) pushed to `{branch}`",
                        "color": 7506394,
                        "fields": [
                            {"name": "Repository", "value": repo.get('full_name', 'unknown'), "inline": True},
                            {"name": "Branch", "value": branch, "inline": True},
                            {"name": "Pusher", "value": pusher.get('name', 'unknown'), "inline": True}
                        ],
                        "timestamp": datetime.utcnow().isoformat() + "Z"
                    }]
                }
                
                # Add commit details
                if commits:
                    commit_list = []
                    for commit in commits[:5]:  # Show max 5 commits
                        commit_msg = commit.get('message', '').split('\n')[0]  # First line only
                        commit_sha = commit.get('id', '')[:7]  # Short SHA
                        commit_list.append(f"`{commit_sha}` {commit_msg}")
                    
                    discord_message["embeds"][0]["fields"].append({
                        "name": "Commits",
                        "value": '\n'.join(commit_list),
                        "inline": False
                    })
            
            # Handle pull request events
            elif event_type == 'pull_request':
                action = payload.get('action', '')
                pr = payload.get('pull_request', {})
                repo = payload.get('repository', {})
                
                colors = {
                    'opened': 3066993,    # Green
                    'closed': 15158332,   # Red
                    'merged': 7506394,    # Purple
                    'reopened': 16776960  # Yellow
                }
                
                discord_message = {
                    "embeds": [{
                        "title": f"🔀 Pull Request {action.title()}",
                        "description": f"**{pr.get('title', 'Unknown')}**",
                        "color": colors.get(action, 7506394),
                        "fields": [
                            {"name": "Repository", "value": repo.get('full_name', 'unknown'), "inline": True},
                            {"name": "Author", "value": pr.get('user', {}).get('login', 'unknown'), "inline": True},
                            {"name": "Branch", "value": f"{pr.get('head', {}).get('ref', 'unknown')} → {pr.get('base', {}).get('ref', 'unknown')}", "inline": True}
                        ],
                        "url": pr.get('html_url', ''),
                        "timestamp": datetime.utcnow().isoformat() + "Z"
                    }]
                }
            
            # Handle issues
            elif event_type == 'issues':
                action = payload.get('action', '')
                issue = payload.get('issue', {})
                repo = payload.get('repository', {})
                
                discord_message = {
                    "embeds": [{
                        "title": f"🐛 Issue {action.title()}",
                        "description": f"**{issue.get('title', 'Unknown')}**",
                        "color": 15158332 if action == 'opened' else 3066993,
                        "fields": [
                            {"name": "Repository", "value": repo.get('full_name', 'unknown'), "inline": True},
                            {"name": "Author", "value": issue.get('user', {}).get('login', 'unknown'), "inline": True},
                            {"name": "Number", "value": f"#{issue.get('number', 'unknown')}", "inline": True}
                        ],
                        "url": issue.get('html_url', ''),
                        "timestamp": datetime.utcnow().isoformat() + "Z"
                    }]
                }
            
            # Handle releases
            elif event_type == 'release':
                action = payload.get('action', '')
                release = payload.get('release', {})
                repo = payload.get('repository', {})
                
                if action == 'published':
                    discord_message = {
                        "embeds": [{
                            "title": f"🚀 New Release: {release.get('tag_name', 'unknown')}",
                            "description": f"**{release.get('name', 'Unknown')}**\n\n{release.get('body', '')[:200]}{'...' if len(release.get('body', '')) > 200 else ''}",
                            "color": 3066993,
                            "fields": [
                                {"name": "Repository", "value": repo.get('full_name', 'unknown'), "inline": True},
                                {"name": "Author", "value": release.get('author', {}).get('login', 'unknown'), "inline": True},
                                {"name": "Tag", "value": release.get('tag_name', 'unknown'), "inline": True}
                            ],
                            "url": release.get('html_url', ''),
                            "timestamp": datetime.utcnow().isoformat() + "Z"
                        }]
                    }
                else:
                    # Skip other release actions
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(b'OK - Release action ignored')
                    return
            
            # Handle other events generically
            else:
                repo = payload.get('repository', {})
                discord_message = {
                    "embeds": [{
                        "title": f"🔔 GitHub Event: {event_type}",
                        "description": f"Event received from **{repo.get('full_name', 'unknown')}**",
                        "color": 7506394,
                        "timestamp": datetime.utcnow().isoformat() + "Z"
                    }]
                }
            
            # Send to Discord
            if DISCORD_WEBHOOK_URL:
                self.send_to_discord(discord_message)
            
            # Respond to GitHub
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "success"}).encode())
            
        except Exception as e:
            print(f"Error processing webhook: {e}")
            self.send_response(500)
            self.end_headers()
            self.wfile.write(json.dumps({"error": str(e)}).encode())
    
    def send_to_discord(self, message):
        """Send message to Discord webhook"""
        try:
            data = json.dumps(message).encode()
            req = urllib.request.Request(
                DISCORD_WEBHOOK_URL,
                data=data,
                headers={'Content-Type': 'application/json'}
            )
            with urllib.request.urlopen(req) as response:
                print(f"Discord response: {response.status}")
        except Exception as e:
            print(f"Failed to send to Discord: {e}")
    
    def do_GET(self):
        """Health check endpoint"""
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps({"status": "healthy", "service": "github-discord-proxy"}).encode())

if __name__ == '__main__':
    if not DISCORD_WEBHOOK_URL:
        print("ERROR: DISCORD_WEBHOOK_URL environment variable is required")
        exit(1)
    
    server = HTTPServer(('0.0.0.0', PORT), GitHubWebhookHandler)
    print(f"GitHub to Discord webhook proxy running on port {PORT}")
    print(f"Discord webhook URL configured: {DISCORD_WEBHOOK_URL[:50]}...")
    server.serve_forever()
