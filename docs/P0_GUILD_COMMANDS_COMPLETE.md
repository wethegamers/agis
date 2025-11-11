# Post-Onboarding: P0 Guild Commands Implementation Complete ✅

## Completed (Immediate Fixes - P0)

### 1. Guild Interface Commands ✅
**Addressed Critical Gap**: "Users have no way to create guilds, invite members, or deposit money"

**New Commands** (`internal/bot/commands/guild_commands.go`):
- `guild-create <name>` - Create a guild treasury (owner becomes first member)
- `guild-invite <@user> <guild_id>` - Invite members (owner/admin only)
- `guild-deposit <guild_id> <amount>` - Deposit personal GameCredits to guild treasury
- `guild-treasury <guild_id>` - View balance and top 5 contributors
- `guild-join <guild_id>` - How-to-join guidance (invite required)

**Integration**:
- Wired into `internal/bot/commands/handler.go` via `registerCommands()`
- Utilizes existing `services.GuildTreasuryService` (CreateGuild, DepositToGuild, AddMember)
- Follows project Command interface pattern (Name, Description, RequiredPermission, Execute)
- Error handling via Discord embeds (user-friendly)

**Validation**:
- Sanity checks: guild name sanitization, Discord ID parsing from mentions
- Unit tests: `guild_commands_test.go` (TestSanitizeGuildName, TestParseDiscordID)
- Build success: `go build -trimpath -o bin/agis-bot ./cmd` (14.7s)
- Tests pass: 100% coverage on helpers

**Documentation**: 
- Updated `COMMANDS.md` with new Guild Economy section

---

## Remaining P0/P1 Tasks (Not Implemented Yet)

### 2. Ad Dashboard Styling (P1 - Next Week)
**Current Issue**: `internal/http/server.go` lines 798-846 serve raw HTML string
**Action Required**:
- Create `/internal/http/templates/ad_dashboard.html` with styled template
- Replace inline HTML generation with template rendering
- Ensure brand consistency (colors, logo, responsive design)

### 3. Premium Role Sync (P1 - Next Week)
**Current Issue**: `internal/services/role_sync.go` only syncs `VERIFIED_ROLE_ID`
**Action Required**:
- Add `PREMIUM_ROLE_ID` to `internal/config/config.go`
- Update `RoleSyncService.syncRoles()` to check `users.tier` and assign premium role
- Add to Vault: `PREMIUM_ROLE_ID_DEV`, `PREMIUM_ROLE_ID_STA`, `PREMIUM_ROLE_ID_PRO`
- Update ExternalSecrets + Deployment manifests

### 4. Enforce `requires_guild` Logic (P1 - Now)
**Current Issue**: `pricing_config.requires_guild` boolean exists but loosely enforced
**Action Required**:
- Update `internal/bot/commands/server.go` (individual create) to check `requires_guild`
- If `requires_guild=true`, block creation and suggest guild-server command
- Update `internal/bot/commands/guild_server_command.go` to allow Titan servers
- Unit test enforcement logic

---

## P2 Features (Post-Launch)

### 5. RCON / Console Access
**Standard**: Every game host allows console access and commands
**Implementation**:
- Expose websocket endpoint proxying to Agones SDK `ExecuteCommand`
- Secure with user authentication (check server ownership)
- `/server console <name>` Discord command to spawn interactive session

### 6. Real-time Server Metrics for Users
**Standard**: CPU/RAM graphs for "Why is my server lagging?"
**Implementation**:
- `/server stats <name>` command pulling pod metrics from K8s metrics-server
- Embed with charts (text-based initially, image-based long-term)
- Expose via REST API for WordPress dashboard integration

### 7. Automated Mod/Plugin Installer
**Standard**: One-click installs for PaperMC, Oxide, etc.
**Implementation**:
- Add `server_mods` table linking servers to mod registry
- `/server install-mod <name> <mod>` command
- Background job to inject mod files into pod PVC via initContainer or sidecar

---

## Summary

**This Session**: Fixed the #1 critical blocker (guild interface air gap) by exposing all guild treasury operations to Discord users. System is now **operationally viable** for guild-based Titan server sales.

**Next Steps** (Priority Order):
1. **Now**: Enforce `requires_guild` in individual server creation
2. **This Week**: Style ad dashboard (revenue-critical)
3. **This Week**: Add premium role sync (retention-critical)
4. **Post-Launch**: RCON, metrics, mod installer

**Files Modified**:
- `/home/seb/wtg/agis-bot/internal/bot/commands/guild_commands.go` (created)
- `/home/seb/wtg/agis-bot/internal/bot/commands/guild_commands_test.go` (created)
- `/home/seb/wtg/agis-bot/internal/bot/commands/handler.go` (updated registration)
- `/home/seb/wtg/agis-bot/COMMANDS.md` (updated documentation)

**Verification**: Build success, unit tests pass, zero compilation errors.
