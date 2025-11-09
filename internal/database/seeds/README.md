# Database Seed Files

This directory contains SQL seed data for populating the database with initial configuration.

## Files

### `pricing_seed.sql`
**Purpose:** Seeds the `pricing_config` table with accurate game server pricing based on real Civo instance costs and profitability targets.

**When to use:**
- **Initial deployment**: Run once after schema creation
- **Price updates**: Re-run with `ON CONFLICT DO UPDATE` to adjust pricing without code deployment
- **New games**: Add new entries and re-run

**How to run:**

```bash
# Local development (DATABASE_URL set in .env)
psql $DATABASE_URL -f internal/database/seeds/pricing_seed.sql

# Staging/Production (from CI/CD or ops terminal)
psql "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}/${DB_NAME}" \
  -f internal/database/seeds/pricing_seed.sql
```

**Safe to re-run:** Yes. Uses `ON CONFLICT DO UPDATE` for idempotent updates.

### Adding New Games

1. Calculate Civo instance size needed (RAM, CPU requirements)
2. Calculate cost: `(Civo hourly cost) / 0.001 USD per GC` × 1.3-1.5 markup
3. Verify margin: `((price - cost) / price) × 100%` should be 25-40%
4. Add entry to `pricing_seed.sql`:
   ```sql
   ('game-slug', 'Display Name', COST_GC, 'g4s.kube.SIZE',
    'Description with economics. Cost: X GC/hr. Margin: Y%.', 
    true, REQUIRES_GUILD_BOOL),
   ```
5. Re-run seed file (safe due to ON CONFLICT clause)

### Pricing Philosophy

**Free-Tier Games** (30-40 GC/hr):
- Target: Attract and retain free users
- Economics: Profitable on ad revenue alone (3000 GC/mo = 75-100 hours)
- Instance: xsmall/small
- Examples: Minecraft, Terraria, DST

**Mid-Tier Games** (90-135 GC/hr):
- Target: Active users, premium subscribers
- Economics: Higher margins, justify premium subscription ROI
- Instance: medium
- Examples: CS2, Valheim, Project Zomboid

**Premium Games** (180-240 GC/hr):
- Target: Premium subscribers or guilds
- Economics: Requires 3x ad multiplier OR guild pooling
- Instance: large/xlarge
- Examples: Rust, Palworld, ARK (vanilla)

**Titan Games** (240+ GC/hr, `requires_guild=true`):
- Target: Guild treasuries only
- Economics: Unprofitable for solo users, profitable with 5+ guild members pooling
- Instance: xlarge/2xlarge
- Examples: ARK: Survival Evolved (heavily modded)

### Updating Prices Without Deployment

Pricing is database-driven, not hardcoded. Update prices via:

**Option 1: SQL (recommended for ops)**
```sql
UPDATE pricing_config 
SET cost_per_hour = 150, description = 'Updated description' 
WHERE game_type = 'rust';
```

**Option 2: Discord command (if admin commands exist)**
```
!pricing update rust 150
!pricing update ark 250
```

**Option 3: Re-run seed file**
```bash
# Modify pricing_seed.sql, then:
psql $DATABASE_URL -f internal/database/seeds/pricing_seed.sql
```

### Monitoring Profitability

Query for games with low margins:

```sql
SELECT 
  game_type,
  display_name,
  cost_per_hour,
  instance_size,
  ROUND(((cost_per_hour::numeric / 1000) - 
    (CASE 
      WHEN instance_size = 'g4s.kube.xsmall' THEN 0.0216
      WHEN instance_size = 'g4s.kube.small' THEN 0.0432
      WHEN instance_size = 'g4s.kube.medium' THEN 0.0864
      WHEN instance_size = 'g4s.kube.large' THEN 0.1728
      WHEN instance_size = 'g4s.kube.xlarge' THEN 0.3456
      ELSE 0.6912
    END)) / (cost_per_hour::numeric / 1000) * 100, 2) AS margin_pct
FROM pricing_config 
WHERE is_active = true
ORDER BY margin_pct ASC;
```

Games with <20% margin should be reviewed and potentially repriced.

---

## Future Seed Files

### `shop_items_seed.sql` (not yet created)
Seeds the `shop_items` table with WTG coin packages:
- 1000 WTG ($0.99)
- 2500 WTG ($1.99)
- 5000 WTG ($3.99) - **Most Popular**
- 11000 WTG ($7.99) - 10% bonus

### `achievements_seed.sql` (not yet created)
Seeds the `achievements` table with milestone achievements:
- First Server Created
- 10 Servers Launched
- 100 Hours Played
- Guild Founder
- etc.
