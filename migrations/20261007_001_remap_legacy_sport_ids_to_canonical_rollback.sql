-- ============================================================================
-- Rollback: 20261007_001_remap_legacy_sport_ids_to_canonical_rollback.sql
-- Namespace: SDA_SPORT_ID_REMAP_V1_ROLLBACK
-- Purpose: Restores legacy local sport IDs if needed during rollback.
-- ============================================================================

START TRANSACTION;

-- Step 1: Move canonical IDs into temp partition (20000 + id)
UPDATE canonical_fixtures SET sport_id = 20004 WHERE sport_id = 4;   -- Ice Hockey
UPDATE canonical_fixtures SET sport_id = 20002 WHERE sport_id = 2;   -- Basketball
UPDATE canonical_fixtures SET sport_id = 20003 WHERE sport_id = 3;   -- Tennis
UPDATE canonical_fixtures SET sport_id = 20005 WHERE sport_id = 5;   -- Volleyball
UPDATE canonical_fixtures SET sport_id = 20006 WHERE sport_id = 6;   -- MMA
UPDATE canonical_fixtures SET sport_id = 20013 WHERE sport_id = 13;  -- Table Tennis
UPDATE canonical_fixtures SET sport_id = 20008 WHERE sport_id = 8;   -- Cricket

-- Step 2: Restore legacy local sport IDs
UPDATE canonical_fixtures SET sport_id = 2  WHERE sport_id = 20004;  -- Ice Hockey -> 2
UPDATE canonical_fixtures SET sport_id = 3  WHERE sport_id = 20002;  -- Basketball -> 3
UPDATE canonical_fixtures SET sport_id = 4  WHERE sport_id = 20003;  -- Tennis -> 4
UPDATE canonical_fixtures SET sport_id = 6  WHERE sport_id = 20005;  -- Volleyball -> 6
UPDATE canonical_fixtures SET sport_id = 9  WHERE sport_id = 20006;  -- MMA -> 9
UPDATE canonical_fixtures SET sport_id = 10 WHERE sport_id = 20013;  -- Table Tennis -> 10
UPDATE canonical_fixtures SET sport_id = 66 WHERE sport_id = 20008;  -- Cricket -> 66

-- Repeat for fixture_mappings
UPDATE fixture_mappings SET sport_id = 20004 WHERE sport_id = 4;
UPDATE fixture_mappings SET sport_id = 20002 WHERE sport_id = 2;
UPDATE fixture_mappings SET sport_id = 20003 WHERE sport_id = 3;
UPDATE fixture_mappings SET sport_id = 20005 WHERE sport_id = 5;
UPDATE fixture_mappings SET sport_id = 20006 WHERE sport_id = 6;
UPDATE fixture_mappings SET sport_id = 20013 WHERE sport_id = 13;
UPDATE fixture_mappings SET sport_id = 20008 WHERE sport_id = 8;

UPDATE fixture_mappings SET sport_id = 2  WHERE sport_id = 20004;
UPDATE fixture_mappings SET sport_id = 3  WHERE sport_id = 20002;
UPDATE fixture_mappings SET sport_id = 4  WHERE sport_id = 20003;
UPDATE fixture_mappings SET sport_id = 6  WHERE sport_id = 20005;
UPDATE fixture_mappings SET sport_id = 9  WHERE sport_id = 20006;
UPDATE fixture_mappings SET sport_id = 10 WHERE sport_id = 20013;
UPDATE fixture_mappings SET sport_id = 66 WHERE sport_id = 20008;

-- Repeat for client_entitlements
UPDATE client_entitlements SET sport_id = 20004 WHERE sport_id = 4;
UPDATE client_entitlements SET sport_id = 20002 WHERE sport_id = 2;
UPDATE client_entitlements SET sport_id = 20003 WHERE sport_id = 3;
UPDATE client_entitlements SET sport_id = 20005 WHERE sport_id = 5;
UPDATE client_entitlements SET sport_id = 20006 WHERE sport_id = 6;
UPDATE client_entitlements SET sport_id = 20013 WHERE sport_id = 13;
UPDATE client_entitlements SET sport_id = 20008 WHERE sport_id = 8;

UPDATE client_entitlements SET sport_id = 2  WHERE sport_id = 20004;
UPDATE client_entitlements SET sport_id = 3  WHERE sport_id = 20002;
UPDATE client_entitlements SET sport_id = 4  WHERE sport_id = 20003;
UPDATE client_entitlements SET sport_id = 6  WHERE sport_id = 20005;
UPDATE client_entitlements SET sport_id = 9  WHERE sport_id = 20006;
UPDATE client_entitlements SET sport_id = 10 WHERE sport_id = 20013;
UPDATE client_entitlements SET sport_id = 66 WHERE sport_id = 20008;

COMMIT;
