-- WTG Shop Items Seed Data
-- Based on WeTheGamers Economy Plan v2.0
-- Dual Currency System: WTG (hard currency) and GC (soft currency)
-- Conversion Rate: 1 WTG = 1000 GC = $1.00 USD

-- ============================================================================
-- WTG COIN PACKAGES (Purchased with real money via payment processor)
-- ============================================================================

INSERT INTO shop_items (item_name, item_type, description, price, currency_type, bonus_amount, is_active)
VALUES
  ('5 WTG Coins', 'wtg_package', 
   'Entry-level WTG package. Perfect for trying out premium features!', 
   5, 'USD', 0, true),
   
  ('11 WTG Coins', 'wtg_package', 
   '10 WTG + 1 Bonus WTG! Best value for casual users.', 
   10, 'USD', 1, true),
   
  ('23 WTG Coins', 'wtg_package', 
   '20 WTG + 3 Bonus WTG! Popular choice for regular players.', 
   20, 'USD', 3, true),
   
  ('60 WTG Coins', 'wtg_package', 
   '50 WTG + 10 Bonus WTG! Maximum value for power users!', 
   50, 'USD', 10, true)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- WTG TO GC CONVERSION (In-app item)
-- ============================================================================

INSERT INTO shop_items (item_name, item_type, description, price, currency_type, bonus_amount, is_active)
VALUES
  ('1000 GameCredits', 'gc_conversion', 
   'Convert 1 WTG to 1000 GameCredits instantly! Skip the grind.', 
   1, 'WTG', 0, true),
   
  ('3000 GameCredits (Server Rent)', 'gc_conversion', 
   'Enough GC to pay for 1 month of a baseline free-tier server! Conversion rate: 3 WTG = 3000 GC', 
   3, 'WTG', 0, true),
   
  ('10000 GameCredits', 'gc_conversion', 
   'Bulk GameCredits package. Great value for multiple servers or upgrades.', 
   10, 'WTG', 0, true)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- FUTURE PREMIUM ITEMS (Cosmetics, Boosts, etc.)
-- ============================================================================

-- These are placeholders for future monetization features
INSERT INTO shop_items (item_name, item_type, description, price, currency_type, bonus_amount, is_active)
VALUES
  ('Server Boost (7 days)', 'boost', 
   '2x performance for your server for 7 days! Faster loading, better uptime.', 
   500, 'GC', 0, false),  -- Not active yet
   
  ('Custom Server Name Color', 'cosmetic', 
   'Make your server stand out in the public lobby with a unique color!', 
   1000, 'GC', 0, false),  -- Not active yet
   
  ('Premium Server Slot', 'server_upgrade', 
   'Unlock an additional server slot beyond the free tier limit.', 
   5, 'WTG', 0, false)  -- Not active yet
ON CONFLICT DO NOTHING;

-- ============================================================================
-- VERIFY SEEDED DATA
-- ============================================================================

-- Count total shop items
SELECT COUNT(*) AS total_items FROM shop_items;

-- Show all WTG packages
SELECT id, item_name, price, currency_type, bonus_amount, is_active 
FROM shop_items 
WHERE item_type = 'wtg_package'
ORDER BY price;

-- Show all GC conversion options
SELECT id, item_name, price, currency_type, is_active 
FROM shop_items 
WHERE item_type = 'gc_conversion'
ORDER BY price;

-- Show summary by currency type
SELECT currency_type, COUNT(*) AS item_count, AVG(price) AS avg_price
FROM shop_items
WHERE is_active = true
GROUP BY currency_type;
