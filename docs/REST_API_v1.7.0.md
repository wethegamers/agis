# AGIS Bot REST API v1.7.0 Documentation

## Base URL
```
https://api.wethegamers.org/api/v1
```

## Authentication
All endpoints require authentication via Bearer token:
```
Authorization: Bearer <discord_id>
```

**Future:** Will use proper API keys from `/api/v1/auth/keys` endpoint.

---

## Endpoints

### Servers

#### List Servers
```http
GET /api/v1/servers
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": 123,
      "name": "my-minecraft-server",
      "game_type": "minecraft",
      "status": "running",
      "address": "10.0.1.5",
      "port": 25565,
      "cost_per_hour": 30,
      "is_public": false,
      "description": "My awesome server",
      "created_at": "2025-11-10T10:00:00Z"
    }
  ]
}
```

#### Get Server
```http
GET /api/v1/servers/:id
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 123,
    "name": "my-minecraft-server",
    "game_type": "minecraft",
    "status": "running",
    ...
  }
}
```

#### Create Server
```http
POST /api/v1/servers
```

**Request Body:**
```json
{
  "game_type": "minecraft",
  "server_name": "my-server",
  "description": "Optional description",
  "is_public": false
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 124,
    "name": "my-server",
    "status": "creating",
    ...
  }
}
```

**Errors:**
- `400` - Invalid request (missing fields, invalid game_type)
- `402` - Insufficient credits
- `500` - Server creation failed

#### Delete Server
```http
DELETE /api/v1/servers/:id
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Server deleted successfully"
  }
}
```

#### Server Actions
```http
POST /api/v1/servers/:id/start
POST /api/v1/servers/:id/stop
POST /api/v1/servers/:id/restart
```

**Status:** 501 Not Implemented (coming soon)

---

### Users

#### Get Current User
```http
GET /api/v1/users/me
```

**Response:**
```json
{
  "success": true,
  "data": {
    "discord_id": "123456789012345678",
    "credits": 1500,
    "wtg_coins": 10,
    "tier": "premium",
    "servers_used": 5,
    "join_date": "2025-01-01T00:00:00Z",
    "subscription_expires": "2025-12-01T00:00:00Z"
  }
}
```

#### Get User Stats
```http
GET /api/v1/users/me/stats
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_servers_created": 15,
    "total_commands_used": 342,
    "total_credits_earned": 5000,
    "total_credits_spent": 3500,
    "rank": 42
  }
}
```

---

### Shop

#### List Packages
```http
GET /api/v1/shop
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "wtg_5",
      "name": "5 WTG Coins",
      "amount_usd": 499,
      "wtg_coins": 5,
      "bonus_coins": 0
    },
    {
      "id": "wtg_11",
      "name": "11 WTG Coins",
      "amount_usd": 999,
      "wtg_coins": 10,
      "bonus_coins": 1
    }
  ]
}
```

---

### Leaderboards

#### Credits Leaderboard
```http
GET /api/v1/leaderboard/credits
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "rank": 1,
      "discord_id": "123456789012345678",
      "credits": 15000,
      "tier": "premium"
    }
  ]
}
```

#### Servers Leaderboard
```http
GET /api/v1/leaderboard/servers
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "rank": 1,
      "discord_id": "123456789012345678",
      "servers": 50,
      "tier": "premium"
    }
  ]
}
```

---

## Error Format

All errors follow this structure:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message"
  }
}
```

**Common Error Codes:**
- `UNAUTHORIZED` - Missing or invalid authorization
- `NOT_FOUND` - Resource not found
- `VALIDATION_ERROR` - Invalid request data
- `INSUFFICIENT_CREDITS` - Not enough credits
- `DATABASE_ERROR` - Internal database error
- `NOT_IMPLEMENTED` - Feature not yet available

---

## Rate Limiting

**Current:** No rate limiting  
**Planned:**
- Free tier: 100 requests/hour
- Premium: 1000 requests/hour
- Enterprise: Unlimited

Rate limit info will be returned in headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1699632000
```

---

## Supported Game Types

| Game Type | Cost/Hour | Status |
|-----------|-----------|--------|
| minecraft | 30 GC | ✅ Active |
| terraria | 35 GC | ✅ Active |
| cs2 | 120 GC | ✅ Active |
| valheim | 120 GC | ✅ Active |
| rust | 220 GC | ✅ Active |
| ark | 240 GC | ✅ Active |
| palworld | 180 GC | ✅ Active |
| dst | 60 GC | ✅ Active |
| gmod | 95 GC | ✅ Active |
| 7d2d | 130 GC | ✅ Active |
| pz | 135 GC | ✅ Active |
| factorio | 100 GC | ✅ Active |
| satisfactory | 240 GC | ✅ Active |
| starbound | 40 GC | ✅ Active |

---

## Examples

### Create a Minecraft Server
```bash
curl -X POST https://api.wethegamers.org/api/v1/servers \
  -H "Authorization: Bearer 123456789012345678" \
  -H "Content-Type: application/json" \
  -d '{
    "game_type": "minecraft",
    "server_name": "epic-survival",
    "description": "Hardcore survival world",
    "is_public": false
  }'
```

### List My Servers
```bash
curl -X GET https://api.wethegamers.org/api/v1/servers \
  -H "Authorization: Bearer 123456789012345678"
```

### Get My Profile
```bash
curl -X GET https://api.wethegamers.org/api/v1/users/me \
  -H "Authorization: Bearer 123456789012345678"
```

---

## Changelog

### v1.7.0 (2025-11-10)
- Initial REST API release
- Server CRUD operations
- User profile endpoints
- Shop package listing
- Leaderboards

### Planned for v1.8.0
- API key management
- Rate limiting
- Server start/stop/restart actions
- Advanced filtering and pagination
- Webhooks for events

---

## Support

- **Discord:** https://discord.gg/wethegamers
- **Documentation:** https://docs.wethegamers.org
- **Issues:** https://github.com/wethegamers/agis-bot/issues
