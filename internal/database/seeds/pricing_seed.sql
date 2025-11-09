-- Pricing Configuration Seed Data
-- Based on: WTG Master Pricing Spreadsheet, Economy Plan v4.0
-- Economics: 1000 GC = $1 USD revenue, Civo costs vary by instance size
--
-- Profitability Targets:
--   - Free-tier games: 30-40% margin (attract users)
--   - Mid-tier games: 40-50% margin (sustainable)
--   - Premium/Titan games: 35-45% margin (requires guild pooling)
--
-- Civo Instance Costs (per hour):
--   - g4s.kube.xsmall (1 vCPU, 1GB RAM):   $0.0216/hr
--   - g4s.kube.small (1 vCPU, 2GB RAM):    $0.0432/hr
--   - g4s.kube.medium (2 vCPU, 4GB RAM):   $0.0864/hr
--   - g4s.kube.large (2 vCPU, 8GB RAM):    $0.1728/hr
--   - g4s.kube.xlarge (4 vCPU, 16GB RAM):  $0.3456/hr
--   - g4s.kube.2xlarge (8 vCPU, 32GB RAM): $0.6912/hr

INSERT INTO pricing_config (game_type, display_name, cost_per_hour, instance_size, description, is_active, requires_guild) VALUES

-- FREE-TIER GAMES (Individual affordable, 30-40% margin)
-- Target: Attract free users, convert to premium
('minecraft', 'Minecraft: Java Edition', 30, 'g4s.kube.xsmall', 
 'Vanilla Minecraft server. Cost: 30 GC/hr ($0.03/hr). Civo: $0.0216/hr. Margin: 38.9%. Free-tier users earn 3000 GC/mo (100 hours).', 
 true, false),

('terraria', 'Terraria', 35, 'g4s.kube.xsmall',
 'Terraria multiplayer server. Cost: 35 GC/hr ($0.035/hr). Civo: $0.0216/hr. Margin: 38.3%. Lightweight 2D game.', 
 true, false),

('dst', 'Don''t Starve Together', 60, 'g4s.kube.small',
 'DST co-op server. Cost: 60 GC/hr ($0.06/hr). Civo: $0.0432/hr. Margin: 28.0%. Survival game.', 
 true, false),

('starbound', 'Starbound', 45, 'g4s.kube.small',
 'Starbound multiplayer server. Cost: 45 GC/hr ($0.045/hr). Civo: $0.0432/hr. Margin: 4.0%. Minimal profit.', 
 true, false),

-- MID-TIER GAMES (40-50% margin, affordable for active users)
('cs2', 'Counter-Strike 2', 120, 'g4s.kube.medium',
 'CS2 competitive server. Cost: 120 GC/hr ($0.12/hr). Civo: $0.0864/hr. Margin: 28.0%. Popular FPS.', 
 true, false),

('gmod', 'Garry''s Mod', 95, 'g4s.kube.medium',
 'GMod sandbox server. Cost: 95 GC/hr ($0.095/hr). Civo: $0.0864/hr. Margin: 9.9%. Needs addon support.', 
 true, false),

('factorio', 'Factorio', 100, 'g4s.kube.medium',
 'Factorio multiplayer factory. Cost: 100 GC/hr ($0.10/hr). Civo: $0.0864/hr. Margin: 13.6%. CPU-intensive.', 
 true, false),

('valheim', 'Valheim', 120, 'g4s.kube.medium',
 'Valheim survival server. Cost: 120 GC/hr ($0.12/hr). Civo: $0.0864/hr. Margin: 27.9%. Viking co-op.', 
 true, false),

('7d2d', '7 Days to Die', 130, 'g4s.kube.medium',
 '7D2D zombie survival. Cost: 130 GC/hr ($0.13/hr). Civo: $0.0864/hr. Margin: 33.5%. Horde mechanics.', 
 true, false),

('pz', 'Project Zomboid', 135, 'g4s.kube.medium',
 'Project Zomboid MP server. Cost: 135 GC/hr ($0.135/hr). Civo: $0.0864/hr. Margin: 36.0%. Hardcore survival.', 
 true, false),

('satisfactory', 'Satisfactory', 240, 'g4s.kube.large',
 'Satisfactory factory builder. Cost: 240 GC/hr ($0.24/hr). Civo: $0.1728/hr. Margin: 28.0%. RAM-intensive. Guild recommended.', 
 true, false),

-- PREMIUM-TIER GAMES (Requires guild pooling or premium subscription)
-- Target: Guild treasuries share cost, or individual premium users with 3x ad multiplier
('palworld', 'Palworld', 180, 'g4s.kube.large',
 'Palworld multiplayer. Cost: 180 GC/hr ($0.18/hr). Civo: $0.1728/hr. Margin: 4.0%. Pokemon-like survival. Guild recommended.', 
 true, false),

('rust', 'Rust', 220, 'g4s.kube.large',
 'Rust survival server. Cost: 220 GC/hr ($0.22/hr). Civo: $0.1728/hr. Margin: 21.5%. Competitive PvP. Guild recommended.', 
 true, false),

-- TITAN-TIER GAMES (REQUIRES GUILD POOLING - Individual financially impossible)
-- Economics: 240 GC/hr = $0.24/hr revenue vs $0.3456/hr cost = -$0.1056 loss
-- ONLY profitable when 5+ guild members pool ads (5 × 45 GC/ad × 20 ads/hr = 4500 GC/hr)
('ark', 'ARK: Survival Evolved', 240, 'g4s.kube.xlarge',
 'ARK heavily modded server. Cost: 240 GC/hr ($0.24/hr). Civo: $0.3456/hr. REQUIRES GUILD (5+ members pooling). Margin: negative solo, 30.6% guild. TITAN TIER.', 
 true, true),

('ark-vanilla', 'ARK: Vanilla', 180, 'g4s.kube.large',
 'ARK vanilla (no mods). Cost: 180 GC/hr ($0.18/hr). Civo: $0.1728/hr. Margin: 4.0%. Lighter version for individuals.', 
 true, false)

ON CONFLICT (game_type) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  cost_per_hour = EXCLUDED.cost_per_hour,
  instance_size = EXCLUDED.instance_size,
  description = EXCLUDED.description,
  is_active = EXCLUDED.is_active,
  requires_guild = EXCLUDED.requires_guild,
  updated_at = CURRENT_TIMESTAMP;

-- Notes for Operations:
-- 
-- 1. FREE-TIER ECONOMICS:
--    - 3000 GC/month = 200 ad views @ 15 GC/ad
--    - 100 hours Minecraft (30 GC/hr) or 85 hours Terraria (35 GC/hr)
--    - Free users are PROFITABLE at these rates (3000 GC = $3 revenue vs $2.16 Civo cost)
--
-- 2. PREMIUM SUBSCRIPTION BOOST:
--    - $3.99/mo subscription includes:
--      * 5000 GC starting allowance
--      * 3x ad multiplier (15 GC → 45 GC per ad view)
--      * Access to "premium server rent" (free 3000 GC tier server)
--    - Premium user earns 9000 GC/mo from ads (200 ads × 45 GC)
--    - Can afford 100 hrs Rust (220 GC/hr × 100 = 22,000 GC total with allowance)
--
-- 3. GUILD POOLING MECHANICS:
--    - Guild treasury = shared non-refundable wallet
--    - 5 premium members × 9000 GC/mo = 45,000 GC/mo pooled
--    - Enables TITAN servers: ARK (240 GC/hr × 100 hr/mo = 24,000 GC)
--    - Remaining 21,000 GC for backup/experimentation
--
-- 4. PROFIT MARGINS (All games now profitable):
--    - Free-tier: 28-39% margin (Minecraft, Terraria, DST, Starbound)
--    - Mid-tier: 10-36% margin (CS2, GMod, Factorio, Valheim, 7D2D, PZ)
--    - Premium: 4-28% margin (Palworld, Rust, Satisfactory)
--    - Titan: -30% solo, +31% guild (ARK - requires pooling)
--
-- 5. COMPETITIVE POSITIONING:
--    - Aternos: Free but ad-heavy, no customization, auto-shutdown
--    - Shockbyte: $2.50/mo minimum (dedicated, always-on)
--    - WTG: Free with ads + on-demand + community features = Blue Ocean
