#!/usr/bin/env python3
"""
GitHub to Discord Webhook Proxy
Converts GitHub webhook events to Discord-compatible messages
"""

import json
import requests
from http.server import HTTPServer, BaseHTTPRequestHandler
import os
from datetime import datetime

DISCORD_WEBHOOK_URL = os.getenv('DISCORD_WEBHOOK_URL', "https://discord.com/api/webhooks/1389136910252904509/m84UqkOAU5UJjnPMWdJ17L5CJ-YzKaSzuD6QSjQw9_RuL-O9abqbLK2_VE2Krsj9wLW_")

class GitHubToDiscordHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        try:
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            github_event = json.loads(post_data.decode('utf-8'))
            
            # Get GitHub event type
            event_type = self.headers.get('X-GitHub-Event', 'unknown')
            
            # Convert GitHub event to Discord message
            discord_message = self.convert_to_discord(github_event, event_type)
            
            if discord_message:
                # Send to Discord
                response = requests.post(DISCORD_WEBHOOK_URL, json=discord_message)
                if response.status_code == 204:
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(b'OK')
                else:
                    print(f"Discord API error: {response.status_code} - {response.text}")
                    self.send_response(500)
                    self.end_headers()
            else:
                # No message to send (unsupported event)
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b'Event ignored')
                
        except Exception as e:
            print(f"Error processing webhook: {e}")
            self.send_response(500)
            self.end_headers()
    
    def convert_to_discord(self, github_event, event_type):
        """Convert GitHub webhook event to Discord message format"""
        
        if event_type == 'ping':
            return {
                "embeds": [{
                    "title": "🏓 GitHub Webhook Connected",
                    "description": "GitHub webhook is now properly configured for agis-bot",
                    "color": 5814783,
                    "timestamp": datetime.utcnow().isoformat() + "Z"
                }]
            }
        
        elif event_type == 'push':
            commits = github_event.get('commits', [])
            ref = github_event.get('ref', '')
            branch = ref.replace('refs/heads/', '') if ref.startswith('refs/heads/') else ref
            
            if not commits:
                return None
                
            return {
                "embeds": [{
                    "title": f"📝 Push to {branch}",
                    "description": f"{len(commits)} commit(s) pushed to {github_event['repository']['full_name']}",
                    "color": 7506394,
                    "fields": [
                        {
                            "name": "Latest Commit",
                            "value": f"[{commits[-1]['id'][:7]}]({commits[-1]['url']}) {commits[-1]['message'][:100]}",
                            "inline": False
                        },
                        {
                            "name": "Author", 
                            "value": commits[-1]['author']['name'],
                            "inline": True
                        },
                        {
                            "name": "Branch",
                            "value": branch,
                            "inline": True
                        }
                    ],
                    "timestamp": datetime.utcnow().isoformat() + "Z"
                }]
            }
        
        elif event_type == 'pull_request':
            pr = github_event['pull_request']
            action = github_event['action']
            
            color_map = {
                'opened': 3066993,    # Green
                'closed': 15158332,   # Red
                'merged': 9442302,    # Purple
                'reopened': 16776960  # Yellow
            }
            
            return {
                "embeds": [{
                    "title": f"🔀 Pull Request {action.title()}",
                    "description": f"#{pr['number']}: {pr['title']}",
                    "url": pr['html_url'],
                    "color": color_map.get(action, 7506394),
                    "fields": [
                        {
                            "name": "Author",
                            "value": pr['user']['login'],
                            "inline": True
                        },
                        {
                            "name": "Branch",
                            "value": f"{pr['head']['ref']} → {pr['base']['ref']}",
                            "inline": True
                        }
                    ],
                    "timestamp": datetime.utcnow().isoformat() + "Z"
                }]
            }
        
        elif event_type == 'release':
            release = github_event['release']
            action = github_event['action']
            
            if action == 'published':
                return {
                    "embeds": [{
                        "title": "🚀 New Release Published",
                        "description": f"{release['tag_name']}: {release['name']}",
                        "url": release['html_url'],
                        "color": 3066993,
                        "fields": [
                            {
                                "name": "Tag",
                                "value": release['tag_name'],
                                "inline": True
                            },
                            {
                                "name": "Author",
                                "value": release['author']['login'],
                                "inline": True
                            }
                        ],
                        "timestamp": datetime.utcnow().isoformat() + "Z"
                    }]
                }
        
        elif event_type == 'issues':
            issue = github_event['issue']
            action = github_event['action']
            
            if action in ['opened', 'closed', 'reopened']:
                color_map = {
                    'opened': 3066993,    # Green
                    'closed': 15158332,   # Red  
                    'reopened': 16776960  # Yellow
                }
                
                return {
                    "embeds": [{
                        "title": f"🐛 Issue {action.title()}",
                        "description": f"#{issue['number']}: {issue['title']}",
                        "url": issue['html_url'],
                        "color": color_map.get(action, 7506394),
                        "fields": [
                            {
                                "name": "Author",
                                "value": issue['user']['login'],
                                "inline": True
                            },
                            {
                                "name": "Labels",
                                "value": ", ".join([label['name'] for label in issue.get('labels', [])]) or "None",
                                "inline": True
                            }
                        ],
                        "timestamp": datetime.utcnow().isoformat() + "Z"
                    }]
                }
        
        # Return None for unsupported events
        return None

if __name__ == "__main__":
    port = int(os.environ.get('PORT', 8080))
    server = HTTPServer(('0.0.0.0', port), GitHubToDiscordHandler)
    print(f"GitHub to Discord webhook proxy running on port {port}")
    server.serve_forever()
