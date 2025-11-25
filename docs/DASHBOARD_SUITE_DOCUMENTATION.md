# Dashboard Suite Documentation

**Created:** November 11, 2025  
**Framework:** Tabler UI v1.0.0-beta20  
**Status:** Multiple versions for iterative refinement

---

## Overview

This document describes the complete dashboard suite for the WeTheGamers (WTG) game server platform. We've created multiple dashboard versions using Tabler UI to support iterative design refinement while building toward the final production interface.

---

## Dashboard Variants

### 1. User Dashboard v1 (Basic) - `user_dashboard_v1.html`

**Purpose:** Clean, minimal server management for end users  
**Target Audience:** Free and premium users  
**Complexity:** Low

#### Features
- **Server Grid View**: Card-based layout with game icons
- **Quick Actions**: Start/stop/restart/delete via dropdowns
- **Server Stats**: Cost per hour, uptime, status indicators
- **Connection Info**: Copy-to-clipboard server addresses
- **Create Server Modal**: Simple form with game selection
- **Quick Stats Bar**: 4 metric cards (servers, credits, uptime, spent)

#### UI Elements
- Animated status indicators (pulse effect for "creating")
- Game-specific color coding (Minecraft green, CS2 orange, etc.)
- Empty state with illustration when no servers exist
- Responsive grid (3 columns desktop, 1 column mobile)

#### Key Interactions
```javascript
// Server actions via REST API
startServer(id)    // POST /api/v1/servers/:id/start
stopServer(id)     // POST /api/v1/servers/:id/stop
restartServer(id)  // POST /api/v1/servers/:id/restart
deleteServer(id)   // DELETE /api/v1/servers/:id
createServer()     // POST /api/v1/servers
```

#### When to Use
- MVP/beta testing
- Users who prefer simplicity
- Mobile-first deployments
- Limited feature set deployments

---

### 2. User Dashboard v2 (Enhanced) - `user_dashboard_v2.html`

**Purpose:** Advanced dashboard with real-time metrics  
**Target Audience:** Power users and premium subscribers  
**Complexity:** Medium-High

#### Features
- **Real-time Metrics**: Live CPU, memory, players, network graphs
- **Enhanced Server Cards**: Console previews, live stats, quick actions
- **Activity Timeline**: Recent server events and actions
- **Chart.js Integration**: Sparkline charts for resource trends
- **Auto-refresh**: Updates every 10 seconds
- **Premium Badge**: Visual indicator for premium users
- **Console Preview**: Last 3-5 log lines per server

#### UI Elements
- Metric cards with border accent colors
- Status pulse animations
- Inline progress bars for CPU/RAM
- Console-style log preview (dark theme)
- Icon-only quick action buttons (40x40px)
- Game-specific badges (Minecraft, CS2, etc.)

#### Real-time Data
```javascript
// Metrics updated via polling
{
  avgCPU: 45,           // Across all servers
  avgMemory: 62,        // Average RAM usage
  totalPlayers: 12,     // Active players sum
  networkIO: 1.2        // MB/s network traffic
}
```

#### Advanced Features
- **WebSocket Support** (planned): Real-time console streaming
- **Player Count**: Live player tracking per server
- **Resource Graphs**: Historical CPU/memory trends
- **Recent Activity**: Timeline of server actions

#### When to Use
- Production deployments
- Premium tier upsell showcase
- Users managing multiple servers
- Monitoring-heavy workflows

---

### 3. Server Control Panel - `server_control_panel.html`

**Purpose:** Dedicated single-server management interface  
**Target Audience:** Server administrators  
**Complexity:** High

#### Features
- **Full Terminal**: Xterm.js integration for console access
- **File Manager**: Browse, upload, edit, delete server files
- **Player Management**: Online players list with kick/ban actions
- **Plugin System**: Enable/disable plugins, view configurations
- **Backup System**: Create, restore, download backups
- **Log Viewer**: Raw server logs with download option
- **Real-time Stats**: CPU, Memory, Network, Disk usage (live)

#### Tab System
1. **Console** - Full terminal emulator with command history
2. **Files** - File browser with upload/download
3. **Players** - Online player list with moderation tools
4. **Plugins** - Plugin management (enable/disable/configure)
5. **Backups** - Backup creation and restoration
6. **Logs** - Raw log file viewer

#### Terminal Integration
```javascript
// Xterm.js with WebSocket
const ws = new WebSocket('ws://host/api/v1/servers/:id/console');
ws.onmessage = (event) => {
  term.writeln(event.data); // Live log streaming
};

// Send commands
ws.send(JSON.stringify({ command: 'help' }));
```

#### File Operations
```javascript
// File manager API (planned)
downloadFile(path)   // GET /api/v1/servers/:id/files?path=...
uploadFile(file)     // POST /api/v1/servers/:id/files
editFile(path)       // PUT /api/v1/servers/:id/files
deleteFile(path)     // DELETE /api/v1/servers/:id/files
```

#### When to Use
- Dedicated server detail pages
- Advanced server administration
- Developer/power user workflows
- Full-featured management needs

---

### 4. Admin Dashboard v2 - `admin_dashboard.html` (existing, to be enhanced)

**Purpose:** Platform-wide analytics for administrators  
**Target Audience:** WTG staff and administrators  
**Complexity:** Medium

#### Current Features (v1)
- Total servers metric
- Active servers count
- Server utilization percentage
- User metrics (total, premium, percentage)

#### Planned Enhancements (v2)
- **Revenue Metrics**: Total revenue, MRR, ARPU
- **User Activity**: DAU/MAU, retention curves
- **Server Distribution**: Heatmap by game type
- **System Health**: Error rates, API response times
- **Fraud Detection**: Suspicious activity alerts
- **Cost Analysis**: Infrastructure costs vs revenue

---

## Design System

### Color Palette

```css
/* Game Type Colors */
.game-minecraft { background: #62a83e; }
.game-cs2       { background: #f59f00; }
.game-terraria  { background: #17a2b8; }
.game-gmod      { background: #f76707; }

/* Status Colors */
.status-running   { color: #2fb344; }
.status-stopped   { color: #6c757d; }
.status-creating  { color: #f59f00; animation: pulse; }
.status-error     { color: #d63939; }

/* Metric Accent Colors */
.metric-cpu     { border-color: #206bc4; }
.metric-memory  { border-color: #d63939; }
.metric-players { border-color: #2fb344; }
.metric-network { border-color: #f59f00; }
```

### Typography

```css
@import url('https://rsms.me/inter/inter.css');

:root {
  --tblr-font-sans-serif: 'Inter Var', -apple-system, BlinkMacSystemFont, ...;
}

body {
  font-feature-settings: "cv03", "cv04", "cv11"; /* Inter font ligatures */
}
```

### Components

#### Server Card
- **Header**: Game icon + name + status badge
- **Body**: Connection info, stats (2-column layout)
- **Footer**: Timestamp + action link
- **Actions**: Dropdown menu for start/stop/delete

#### Metric Card
- **Icon**: Avatar with background color
- **Label**: Metric name
- **Value**: Large heading (h1-h3)
- **Subtext**: Secondary information
- **Chart** (optional): Sparkline graph

#### Status Badge
```html
<span class="badge bg-success">running</span>
<span class="badge bg-warning status-pulse">creating</span>
<span class="badge bg-secondary">stopped</span>
<span class="badge bg-danger">error</span>
```

---

## API Integration

### Authentication
All dashboards use simplified Bearer token authentication:

```javascript
headers: {
  'Authorization': 'Bearer ' + localStorage.getItem('token')
}
```

**Note:** Currently using Discord ID as token. Migrate to proper API keys per REST API implementation plan.

### Endpoints Used

| Endpoint | Method | Dashboard | Purpose |
|----------|--------|-----------|---------|
| `/api/v1/servers` | GET | All | List user servers |
| `/api/v1/servers` | POST | v1, v2 | Create server |
| `/api/v1/servers/:id` | GET | Control Panel | Get server details |
| `/api/v1/servers/:id/start` | POST | All | Start server |
| `/api/v1/servers/:id/stop` | POST | All | Stop server |
| `/api/v1/servers/:id/restart` | POST | All | Restart server |
| `/api/v1/servers/:id` | DELETE | All | Delete server |
| `/api/v1/users/me` | GET | All | Get current user |
| `/api/v1/users/me/stats` | GET | v2 | Get user stats |

### Data Models

#### Server Object
```javascript
{
  id: 42,
  name: "my-minecraft-server",
  game_type: "minecraft",
  status: "running",
  address: "play.example.com",
  port: 25565,
  cost_per_hour: 5,
  uptime: "2h 34m",
  cpu: 45,              // Percentage
  memory: 62,           // Percentage
  players: 3,           // Current
  max_players: 20,
  created_at: "2h ago",
  kubernetes_uid: "abc123...",
  agones_status: "Ready"
}
```

#### User Object
```javascript
{
  discord_id: "123456789",
  username: "Player123",
  credits: 150,
  tier: "premium",
  servers_used: 3,
  total_uptime: "48h 20m",
  total_spent: 450
}
```

---

## Responsive Design

### Breakpoints
- **Mobile**: < 768px (1 column layout)
- **Tablet**: 768px - 1024px (2 columns)
- **Desktop**: > 1024px (3-4 columns)

### Mobile Optimizations
- Collapsible navbar
- Stacked metric cards
- Touch-friendly buttons (min 44x44px)
- Simplified dropdowns
- Bottom sheet modals (on mobile)

---

## Performance Considerations

### Auto-refresh Intervals
- **v1**: 30 seconds (full page reload)
- **v2**: 10 seconds (API polling)
- **Control Panel**: 5 seconds (stats), real-time (console via WebSocket)

### Optimization Strategies
1. **Lazy Load**: Don't render hidden tabs until clicked
2. **Pagination**: Limit servers per page (20-50)
3. **Debounce**: Rate-limit user actions (1 req/2s)
4. **Cache**: LocalStorage for user preferences
5. **Compress**: Minify CSS/JS in production

### Bundle Sizes
- **v1**: ~150KB (Tabler CSS + minimal JS)
- **v2**: ~280KB (+ Chart.js)
- **Control Panel**: ~450KB (+ Xterm.js + FitAddon)

---

## Future Enhancements

### Phase 2 (v1.7.1)
- [ ] WebSocket support for real-time updates
- [ ] Toast notifications (success/error messages)
- [ ] Keyboard shortcuts (Cmd+K for search)
- [ ] Dark mode toggle
- [ ] Server templates (save/restore configs)

### Phase 3 (v1.8.0)
- [ ] Drag-and-drop file uploads
- [ ] Inline file editor (Monaco Editor)
- [ ] RCON command library
- [ ] Server cloning
- [ ] Batch operations (start/stop multiple)

### Phase 4 (v2.0.0)
- [ ] Mobile app (React Native)
- [ ] Desktop app (Electron)
- [ ] VR dashboard (Three.js)
- [ ] Voice commands (Web Speech API)

---

## Deployment

### Production Checklist
- [ ] Minify CSS/JS assets
- [ ] Enable CDN caching (Tabler from CDN)
- [ ] Add CSP headers
- [ ] Enable gzip compression
- [ ] Set proper cache headers
- [ ] Add error boundary components
- [ ] Configure analytics (Plausible/Google Analytics)
- [ ] Add monitoring (Sentry for errors)

### Environment Variables
```bash
# API endpoint
API_BASE_URL=https://api.wethegamers.org

# WebSocket endpoint
WS_BASE_URL=wss://api.wethegamers.org

# CDN assets (Tabler)
TABLER_CSS_URL=https://cdn.jsdelivr.net/npm/@tabler/core@1.0.0-beta20/dist/css/tabler.min.css
TABLER_JS_URL=https://cdn.jsdelivr.net/npm/@tabler/core@1.0.0-beta20/dist/js/tabler.min.js
```

---

## Testing

### Browser Support
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

### Test Scenarios
1. **Server Lifecycle**: Create → Start → Stop → Restart → Delete
2. **Error Handling**: Network failures, API errors, validation
3. **Performance**: 50+ servers, rapid actions, long sessions
4. **Mobile**: Touch gestures, responsive layout, offline mode
5. **Accessibility**: Screen readers, keyboard nav, contrast ratios

### Manual Testing Script
```bash
# 1. Load dashboard
open https://dashboard.wethegamers.org

# 2. Create server
# - Select Minecraft
# - Name: "test-server-1"
# - Click "Deploy Server"

# 3. Wait for server to start (watch status change)
# - pending → creating → starting → ready

# 4. Test actions
# - Copy server address
# - Open console
# - Restart server
# - Stop server

# 5. Delete server
# - Confirm deletion
# - Verify removed from list
```

---

## Conclusion

This dashboard suite provides multiple UI options to support iterative design refinement. Start with **v1** for MVP, graduate to **v2** for production, and use the **Control Panel** for power users.

All dashboards share the same REST API backend, making it easy to A/B test variants or offer multiple UI skins to different user segments.

**Next Steps:**
1. Integrate dashboards with existing HTTP server
2. Add route handlers for new templates
3. Implement missing API endpoints (files, players, plugins)
4. Add WebSocket support for real-time features
5. User testing to determine final design direction
