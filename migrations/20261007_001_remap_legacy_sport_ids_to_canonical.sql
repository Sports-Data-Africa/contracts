-- ============================================================================
-- Migration: 20261007_001_remap_legacy_sport_ids_to_canonical.sql
-- Namespace: SDA_SPORT_ID_REMAP_V1
-- Purpose: Remap legacy provider/local sport IDs to Platform Canonical IDs (1-13).
-- Safety: Idempotent two-phase offset migration to prevent collision overwrites.
-- Invariant: NEVER RUN AUTOMATICALLY ON SHARED/PRODUCTION DATABASES.
-- ============================================================================

-- DRY-RUN VERIFICATION QUERY (Run this before applying migration to audit record counts):
/*
SELECT 'canonical_fixtures' AS table_name, sport_id, COUNT(*) AS count
FROM canonical_fixtures
WHERE sport_id IN (2, 3, 4, 6, 7, 9, 10, 66)
GROUP BY sport_id
UNION ALL
SELECT 'fixture_mappings', sport_id, COUNT(*)
FROM fixture_mappings
WHERE sport_id IN (2, 3, 4, 6, 7, 9, 10, 66)
GROUP BY sport_id
UNION ALL
SELECT 'client_entitlements', sport_id, COUNT(*)
FROM client_entitlements
WHERE sport_id IN (2, 3, 4, 6, 7, 9, 10, 66)
GROUP BY sport_id;
*/

START TRANSACTION;

-- ----------------------------------------------------------------------------
-- STEP 1: Temp Offset Phase (10000 + id)
-- Isolates existing legacy records so subsequent canonical updates do not collide.
-- ----------------------------------------------------------------------------

-- Table: canonical_fixtures
UPDATE canonical_fixtures SET sport_id = 10002 WHERE sport_id = 2;   -- Legacy Ice Hockey (2 -> 4)
UPDATE canonical_fixtures SET sport_id = 10003 WHERE sport_id = 3;   -- Legacy Basketball (3 -> 2)
UPDATE canonical_fixtures SET sport_id = 10004 WHERE sport_id = 4;   -- Legacy Tennis     (4 -> 3)
UPDATE canonical_fixtures SET sport_id = 10006 WHERE sport_id = 6;   -- Legacy Volleyball (6 -> 5)
UPDATE canonical_fixtures SET sport_id = 10009 WHERE sport_id = 9;   -- Legacy MMA        (9 -> 6)
UPDATE canonical_fixtures SET sport_id = 10010 WHERE sport_id = 10;  -- Legacy Table Tennis (10 -> 13)
UPDATE canonical_fixtures SET sport_id = 10066 WHERE sport_id = 66;  -- Legacy Cricket    (66 -> 8)

-- Table: fixture_mappings
UPDATE fixture_mappings SET sport_id = 10002 WHERE sport_id = 2;
UPDATE fixture_mappings SET sport_id = 10003 WHERE sport_id = 3;
UPDATE fixture_mappings SET sport_id = 10004 WHERE sport_id = 4;
UPDATE fixture_mappings SET sport_id = 10006 WHERE sport_id = 6;
UPDATE fixture_mappings SET sport_id = 10009 WHERE sport_id = 9;
UPDATE fixture_mappings SET sport_id = 10010 WHERE sport_id = 10;
UPDATE fixture_mappings SET sport_id = 10066 WHERE sport_id = 66;

-- Table: client_entitlements
UPDATE client_entitlements SET sport_id = 10002 WHERE sport_id = 2;
UPDATE client_entitlements SET sport_id = 10003 WHERE sport_id = 3;
UPDATE client_entitlements SET sport_id = 10004 WHERE sport_id = 4;
UPDATE client_entitlements SET sport_id = 10006 WHERE sport_id = 6;
UPDATE client_entitlements SET sport_id = 10009 WHERE sport_id = 9;
UPDATE client_entitlements SET sport_id = 10010 WHERE sport_id = 10;
UPDATE client_entitlements SET sport_id = 10066 WHERE sport_id = 66;

-- ----------------------------------------------------------------------------
-- STEP 2: Canonical Remap Phase
-- Maps from temp partition into authoritative Platform Canonical IDs.
-- ----------------------------------------------------------------------------

-- Table: canonical_fixtures
UPDATE canonical_fixtures SET sport_id = 4  WHERE sport_id = 10002; -- Ice Hockey: 4
UPDATE canonical_fixtures SET sport_id = 2  WHERE sport_id = 10003; -- Basketball: 2
UPDATE canonical_fixtures SET sport_id = 3  WHERE sport_id = 10004; -- Tennis: 3
UPDATE canonical_fixtures SET sport_id = 5  WHERE sport_id = 10006; -- Volleyball: 5
UPDATE canonical_fixtures SET sport_id = 6  WHERE sport_id = 10009; -- MMA: 6
UPDATE canonical_fixtures SET sport_id = 13 WHERE sport_id = 10010; -- Table Tennis: 13
UPDATE canonical_fixtures SET sport_id = 8  WHERE sport_id = 10066; -- Cricket: 8

-- Table: fixture_mappings
UPDATE fixture_mappings SET sport_id = 4  WHERE sport_id = 10002;
UPDATE fixture_mappings SET sport_id = 2  WHERE sport_id = 10003;
UPDATE fixture_mappings SET sport_id = 3  WHERE sport_id = 10004;
UPDATE fixture_mappings SET sport_id = 5  WHERE sport_id = 10006;
UPDATE fixture_mappings SET sport_id = 6  WHERE sport_id = 10009;
UPDATE fixture_mappings SET sport_id = 13 WHERE sport_id = 10010;
UPDATE fixture_mappings SET sport_id = 8  WHERE sport_id = 10066;

-- Table: client_entitlements
UPDATE client_entitlements SET sport_id = 4  WHERE sport_id = 10002;
UPDATE client_entitlements SET sport_id = 2  WHERE sport_id = 10003;
UPDATE client_entitlements SET sport_id = 3  WHERE sport_id = 10004;
UPDATE client_entitlements SET sport_id = 5  WHERE sport_id = 10006;
UPDATE client_entitlements SET sport_id = 6  WHERE sport_id = 10009;
UPDATE client_entitlements SET sport_id = 13 WHERE sport_id = 10010;
UPDATE client_entitlements SET sport_id = 8  WHERE sport_id = 10066;

COMMIT;

-- POST-MIGRATION AUDIT QUERY:
-- Verify no records remain in temporary 10000+ partition or legacy Cricket 66:
/*
SELECT 'canonical_fixtures' AS table_name, sport_id, COUNT(*) AS count
FROM canonical_fixtures
WHERE sport_id > 10000 OR sport_id = 66
GROUP BY sport_id;
*/
