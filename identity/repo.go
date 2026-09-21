package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
)

var (
	ErrFixtureNotFound   = errors.New("FIXTURE_NOT_FOUND")
	ErrAliasNotFound     = errors.New("ALIAS_NOT_FOUND")
	ErrIdentityConflict  = errors.New("IDENTITY_CONFLICT_DETECTED")
	ErrAmbiguousFixture  = errors.New("AMBIGUOUS_FIXTURE_MATCH")
)

type Repository interface {
	GetCanonicalFixture(ctx context.Context, canonicalID string) (*CanonicalFixture, error)
	GetCanonicalFixtureByNaturalHash(ctx context.Context, hash string) (*CanonicalFixture, error)
	GetOrCreateCanonicalFixture(ctx context.Context, sportID int, homeTeam, awayTeam string, startTime time.Time, compID int64, compName string) (*CanonicalFixture, error)
	CreateJITCanonicalFixtureTx(ctx context.Context, candidate canonical.NaturalCandidate, provider, providerFixtureID string, mappingType canonical.MappingType, source string) (*CanonicalFixture, *FixtureAlias, error)
	ResolveCandidate(ctx context.Context, candidate canonical.NaturalCandidate) (ResolutionOutcome, *CanonicalFixture, error)
	GetAlias(ctx context.Context, provider, providerFixtureID string) (*FixtureAlias, error)
	AttachAlias(ctx context.Context, canonicalID, provider, providerFixtureID string, mappingType canonical.MappingType, source string) (*FixtureAlias, error)
	GetAliasesForCanonical(ctx context.Context, canonicalID string) ([]FixtureAlias, error)
	RecordConflict(ctx context.Context, conflict *IdentityConflict) error
	InitSchema(ctx context.Context) error
}

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) InitSchema(ctx context.Context) error {
	if r.db == nil {
		return nil
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS canonical_sequence_counter (
			id INT PRIMARY KEY,
			current_val BIGINT NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,

		`INSERT IGNORE INTO canonical_sequence_counter (id, current_val) VALUES (1, 100000);`,

		`CREATE TABLE IF NOT EXISTS canonical_fixtures (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			canonical_fixture_id VARCHAR(64) NOT NULL,
			sequence_number BIGINT NOT NULL,
			sport_id INT NOT NULL,
			sport_code VARCHAR(8) NOT NULL,
			home_team VARCHAR(255) NOT NULL,
			away_team VARCHAR(255) NOT NULL,
			scheduled_start DATETIME NOT NULL,
			competition_id BIGINT NOT NULL DEFAULT 0,
			competition_name VARCHAR(255) NOT NULL DEFAULT '',
			gender VARCHAR(16) NOT NULL DEFAULT 'UNKNOWN',
			age_category VARCHAR(16) NOT NULL DEFAULT 'UNKNOWN',
			team_category VARCHAR(16) NOT NULL DEFAULT 'UNKNOWN',
			natural_identity_hash VARCHAR(64) NOT NULL,
			identity_state VARCHAR(32) NOT NULL DEFAULT 'IDENTIFIED',
			verification_state VARCHAR(32) NOT NULL DEFAULT 'PENDING',
			fixture_state VARCHAR(32) NOT NULL DEFAULT 'NOT_STARTED',
			betting_state VARCHAR(32) NOT NULL DEFAULT 'OPEN',
			settlement_state VARCHAR(32) NOT NULL DEFAULT 'PENDING',
			settlement_capable TINYINT(1) NOT NULL DEFAULT 1,
			result_resolution_key VARCHAR(128) NOT NULL DEFAULT '',
			created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			UNIQUE KEY uq_canonical_fixture_id (canonical_fixture_id),
			UNIQUE KEY uq_sequence_number (sequence_number),
			UNIQUE KEY uq_natural_identity (natural_identity_hash),
			INDEX idx_sport_fixture_state (sport_id, fixture_state),
			INDEX idx_verification_state (verification_state),
			INDEX idx_scheduled_start (scheduled_start),
			INDEX idx_betting_state (betting_state)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS fixture_aliases (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			canonical_fixture_id VARCHAR(64) NOT NULL,
			provider VARCHAR(64) NOT NULL,
			provider_fixture_id VARCHAR(128) NOT NULL,
			mapping_type VARCHAR(32) NOT NULL DEFAULT 'PREMATCH',
			source VARCHAR(64) NOT NULL DEFAULT '',
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			result_resolution_path VARCHAR(255) NOT NULL DEFAULT '',
			first_seen_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			last_seen_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			metadata JSON NULL,
			created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
			UNIQUE KEY uq_provider_alias (provider, provider_fixture_id),
			INDEX idx_canonical_alias (canonical_fixture_id),
			INDEX idx_mapping_type (mapping_type),
			INDEX idx_is_active (is_active)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS identity_conflicts (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			conflict_type VARCHAR(64) NOT NULL,
			provider VARCHAR(64) NOT NULL,
			provider_fixture_id VARCHAR(128) NOT NULL,
			existing_canonical_id VARCHAR(64) NOT NULL,
			proposed_canonical_id VARCHAR(64) NOT NULL,
			evidence JSON NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'UNRESOLVED',
			created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			resolved_at DATETIME(3) NULL,
			INDEX idx_conflict_status (status),
			INDEX idx_conflict_provider (provider, provider_fixture_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
	}

	for _, q := range queries {
		if _, err := r.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("init schema error: %w", err)
		}
	}

	// Guarantee all projection & classification columns exist on pre-existing tables
	cols := []struct {
		table string
		col   string
		def   string
	}{
		{"canonical_fixtures", "competition_id", "BIGINT NOT NULL DEFAULT 0"},
		{"canonical_fixtures", "competition_name", "VARCHAR(255) NOT NULL DEFAULT ''"},
		{"canonical_fixtures", "gender", "VARCHAR(16) NOT NULL DEFAULT 'UNKNOWN'"},
		{"canonical_fixtures", "age_category", "VARCHAR(16) NOT NULL DEFAULT 'UNKNOWN'"},
		{"canonical_fixtures", "team_category", "VARCHAR(16) NOT NULL DEFAULT 'UNKNOWN'"},
		{"canonical_fixtures", "betting_state", "VARCHAR(32) NOT NULL DEFAULT 'OPEN'"},
		{"canonical_fixtures", "settlement_state", "VARCHAR(32) NOT NULL DEFAULT 'PENDING'"},
		{"canonical_fixtures", "settlement_capable", "TINYINT(1) NOT NULL DEFAULT 1"},
		{"canonical_fixtures", "result_resolution_key", "VARCHAR(128) NOT NULL DEFAULT ''"},
		{"fixture_aliases", "is_active", "TINYINT(1) NOT NULL DEFAULT 1"},
		{"fixture_aliases", "result_resolution_path", "VARCHAR(255) NOT NULL DEFAULT ''"},
	}

	for _, c := range cols {
		ensureTableColumn(ctx, r.db, c.table, c.col, c.def)
	}

	indexes := []struct {
		table   string
		idxName string
		idxCols string
	}{
		{"canonical_fixtures", "idx_betting_state", "(betting_state)"},
		{"fixture_aliases", "idx_is_active", "(is_active)"},
	}

	for _, idx := range indexes {
		ensureTableIndex(ctx, r.db, idx.table, idx.idxName, idx.idxCols)
	}

	return nil
}

func ensureTableColumn(ctx context.Context, db *sql.DB, table, col, colDef string) {
	if db == nil {
		return
	}
	var count int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() 
		  AND TABLE_NAME = ? 
		  AND COLUMN_NAME = ?
	`, table, col).Scan(&count)
	if count == 0 {
		alterSQL := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", table, col, colDef)
		_, _ = db.ExecContext(ctx, alterSQL)
	}
}

func ensureTableIndex(ctx context.Context, db *sql.DB, table, idxName, idxCols string) {
	if db == nil {
		return
	}
	var count int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM information_schema.STATISTICS 
		WHERE TABLE_SCHEMA = DATABASE() 
		  AND TABLE_NAME = ? 
		  AND INDEX_NAME = ?
	`, table, idxName).Scan(&count)
	if count == 0 {
		alterSQL := fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `%s` %s", table, idxName, idxCols)
		_, _ = db.ExecContext(ctx, alterSQL)
	}
}

func isUnknownColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Error 1054") || strings.Contains(msg, "Unknown column")
}

func (r *SQLRepository) GetCanonicalFixture(ctx context.Context, canonicalID string) (*CanonicalFixture, error) {
	if r.db == nil {
		return nil, ErrFixtureNotFound
	}
	query := `
		SELECT id, canonical_fixture_id, sequence_number, sport_id, sport_code,
		       home_team, away_team, scheduled_start, competition_id, competition_name,
		       gender, age_category, team_category,
		       natural_identity_hash, identity_state, verification_state, fixture_state,
		       betting_state, settlement_state, settlement_capable, result_resolution_key,
		       created_at, updated_at
		FROM canonical_fixtures
		WHERE canonical_fixture_id = ?
		LIMIT 1
	`
	var f CanonicalFixture
	var idState, verState, fixState, betState, setState string
	var setCapable bool
	err := r.db.QueryRowContext(ctx, query, canonicalID).Scan(
		&f.ID, &f.CanonicalFixtureID, &f.SequenceNumber, &f.SportID, &f.SportCode,
		&f.HomeTeam, &f.AwayTeam, &f.ScheduledStart, &f.CompetitionID, &f.CompetitionName,
		&f.Gender, &f.AgeCategory, &f.TeamCategory,
		&f.NaturalIdentityHash, &idState, &verState, &fixState,
		&betState, &setState, &setCapable, &f.ResultResolutionKey,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrFixtureNotFound
	}
	if err != nil {
		return nil, err
	}
	f.IdentityState = canonical.IdentityState(idState)
	f.VerificationState = canonical.VerificationState(verState)
	f.FixtureState = canonical.FixtureLifecycleState(fixState)
	f.BettingState = canonical.BettingState(betState)
	f.SettlementState = canonical.SettlementLifecycleState(setState)
	f.SettlementCapable = setCapable
	return &f, nil
}

func (r *SQLRepository) GetCanonicalFixtureByNaturalHash(ctx context.Context, hash string) (*CanonicalFixture, error) {
	if r.db == nil {
		return nil, ErrFixtureNotFound
	}
	query := `
		SELECT id, canonical_fixture_id, sequence_number, sport_id, sport_code,
		       home_team, away_team, scheduled_start, competition_id, competition_name,
		       gender, age_category, team_category,
		       natural_identity_hash, identity_state, verification_state, fixture_state,
		       betting_state, settlement_state, settlement_capable, result_resolution_key,
		       created_at, updated_at
		FROM canonical_fixtures
		WHERE natural_identity_hash = ?
		LIMIT 1
	`
	var f CanonicalFixture
	var idState, verState, fixState, betState, setState string
	var setCapable bool
	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&f.ID, &f.CanonicalFixtureID, &f.SequenceNumber, &f.SportID, &f.SportCode,
		&f.HomeTeam, &f.AwayTeam, &f.ScheduledStart, &f.CompetitionID, &f.CompetitionName,
		&f.Gender, &f.AgeCategory, &f.TeamCategory,
		&f.NaturalIdentityHash, &idState, &verState, &fixState,
		&betState, &setState, &setCapable, &f.ResultResolutionKey,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrFixtureNotFound
	}
	if err != nil {
		return nil, err
	}
	f.IdentityState = canonical.IdentityState(idState)
	f.VerificationState = canonical.VerificationState(verState)
	f.FixtureState = canonical.FixtureLifecycleState(fixState)
	f.BettingState = canonical.BettingState(betState)
	f.SettlementState = canonical.SettlementLifecycleState(setState)
	f.SettlementCapable = setCapable
	return &f, nil
}

func (r *SQLRepository) GetOrCreateCanonicalFixture(ctx context.Context, sportID int, homeTeam, awayTeam string, startTime time.Time, compID int64, compName string) (*CanonicalFixture, error) {
	if strings.TrimSpace(homeTeam) == "" || strings.TrimSpace(awayTeam) == "" {
		return nil, errors.New("home_team and away_team are required")
	}
	if startTime.IsZero() {
		return nil, errors.New("scheduled_start is required")
	}

	candidate := canonical.NaturalCandidate{
		SportID:         sportID,
		HomeTeam:        homeTeam,
		AwayTeam:        awayTeam,
		ScheduledStart:  startTime,
		CompetitionID:   fmt.Sprintf("%d", compID),
		CompetitionName: compName,
	}
	candidate.CleanAndClassifyCandidate()
	hash := candidate.NaturalKey()

	// Check if fixture already exists
	existing, err := r.GetCanonicalFixtureByNaturalHash(ctx, hash)
	if err == nil && existing != nil {
		return existing, nil
	}

	if r.db == nil {
		// In-memory fallback if no database connection
		seq := time.Now().Unix()%900000 + 100000
		cid, _ := canonical.FormatCanonicalFixtureID(sportID, seq)
		now := time.Now().UTC()
		return &CanonicalFixture{
			ID:                  1,
			CanonicalFixtureID:  cid,
			SequenceNumber:      seq,
			SportID:             sportID,
			SportCode:           canonical.SportIDToCode[sportID],
			HomeTeam:            homeTeam,
			AwayTeam:            awayTeam,
			ScheduledStart:      startTime,
			CompetitionID:       compID,
			CompetitionName:     compName,
			Gender:              candidate.Gender,
			AgeCategory:         candidate.AgeCategory,
			TeamCategory:        candidate.TeamCategory,
			NaturalIdentityHash: hash,
			IdentityState:       canonical.IdentityIdentified,
			VerificationState:   canonical.VerificationPending,
			FixtureState:        canonical.FixtureNotStarted,
			BettingState:        canonical.BettingOpen,
			SettlementState:     canonical.SettlementPending,
			SettlementCapable:   true,
			ResultResolutionKey: hash,
			CreatedAt:           now,
			UpdatedAt:           now,
		}, nil
	}

	// Begin transactional creation with sequence generation
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Double check within transaction
	var existingID int64
	var existingCanonicalID string
	err = tx.QueryRowContext(ctx, "SELECT id, canonical_fixture_id FROM canonical_fixtures WHERE natural_identity_hash = ? LIMIT 1", hash).Scan(&existingID, &existingCanonicalID)
	if err == nil && existingCanonicalID != "" {
		_ = tx.Rollback()
		return r.GetCanonicalFixture(ctx, existingCanonicalID)
	}

	// 2. Increment sequence atomically
	var nextSeq int64
	_, err = tx.ExecContext(ctx, "UPDATE canonical_sequence_counter SET current_val = current_val + 1 WHERE id = 1")
	if err != nil {
		return nil, fmt.Errorf("increment sequence: %w", err)
	}
	err = tx.QueryRowContext(ctx, "SELECT current_val FROM canonical_sequence_counter WHERE id = 1").Scan(&nextSeq)
	if err != nil {
		return nil, fmt.Errorf("query sequence: %w", err)
	}

	cid, err := canonical.FormatCanonicalFixtureID(sportID, nextSeq)
	if err != nil {
		return nil, fmt.Errorf("format canonical ID: %w", err)
	}

	sportCode := canonical.SportIDToCode[sportID]
	now := time.Now().UTC()

	// 3. Insert canonical fixture
	insertQuery := `
		INSERT INTO canonical_fixtures (
			canonical_fixture_id, sequence_number, sport_id, sport_code,
			home_team, away_team, scheduled_start, competition_id, competition_name,
			gender, age_category, team_category,
			natural_identity_hash, identity_state, verification_state, fixture_state,
			betting_state, settlement_state, settlement_capable, result_resolution_key,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)
	`
	res, err := tx.ExecContext(ctx, insertQuery,
		cid, nextSeq, sportID, sportCode,
		homeTeam, awayTeam, startTime.UTC(), compID, compName,
		candidate.Gender, candidate.AgeCategory, candidate.TeamCategory,
		hash, string(canonical.IdentityIdentified), string(canonical.VerificationPending), string(canonical.FixtureNotStarted),
		string(canonical.BettingOpen), string(canonical.SettlementPending), hash,
		now, now,
	)
	if err != nil {
		_ = tx.Rollback()
		if isUnknownColumnError(err) {
			_ = r.InitSchema(ctx)
			return r.GetOrCreateCanonicalFixture(ctx, sportID, homeTeam, awayTeam, startTime, compID, compName)
		}
		// Race condition handling: if another concurrent worker inserted same hash
		if existingRace, rErr := r.GetCanonicalFixtureByNaturalHash(ctx, hash); rErr == nil && existingRace != nil {
			return existingRace, nil
		}
		return nil, fmt.Errorf("insert canonical fixture: %w", err)
	}

	lastID, _ := res.LastInsertId()
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &CanonicalFixture{
		ID:                  lastID,
		CanonicalFixtureID:  cid,
		SequenceNumber:      nextSeq,
		SportID:             sportID,
		SportCode:           sportCode,
		HomeTeam:            homeTeam,
		AwayTeam:            awayTeam,
		ScheduledStart:      startTime.UTC(),
		CompetitionID:       compID,
		CompetitionName:     compName,
		Gender:              candidate.Gender,
		AgeCategory:         candidate.AgeCategory,
		TeamCategory:        candidate.TeamCategory,
		NaturalIdentityHash: hash,
		IdentityState:       canonical.IdentityIdentified,
		VerificationState:   canonical.VerificationPending,
		FixtureState:        canonical.FixtureNotStarted,
		BettingState:        canonical.BettingOpen,
		SettlementState:     canonical.SettlementPending,
		SettlementCapable:   true,
		ResultResolutionKey: hash,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

// CreateJITCanonicalFixtureTx creates a canonical fixture and binds the provider alias
// in a single atomic database transaction, guaranteeing zero duplicate canonical records under concurrency.
func (r *SQLRepository) CreateJITCanonicalFixtureTx(
	ctx context.Context,
	candidate canonical.NaturalCandidate,
	provider, providerFixtureID string,
	mappingType canonical.MappingType,
	source string,
) (*CanonicalFixture, *FixtureAlias, error) {
	if provider == "" || providerFixtureID == "" {
		return nil, nil, errors.New("provider and providerFixtureID are required")
	}
	candidate.CleanAndClassifyCandidate()
	hash := candidate.NaturalKey()

	// 1. Check if provider alias already exists
	alias, err := r.GetAlias(ctx, provider, providerFixtureID)
	if err == nil && alias != nil {
		f, fErr := r.GetCanonicalFixture(ctx, alias.CanonicalFixtureID)
		return f, alias, fErr
	}

	// 2. Check if natural identity key already exists in DB
	existing, err := r.GetCanonicalFixtureByNaturalHash(ctx, hash)
	if err == nil && existing != nil {
		newAlias, aErr := r.AttachAlias(ctx, existing.CanonicalFixtureID, provider, providerFixtureID, mappingType, source)
		return existing, newAlias, aErr
	}

	if r.db == nil {
		// In-memory fallback
		seq := time.Now().Unix()%900000 + 100000
		cid, _ := canonical.FormatCanonicalFixtureID(candidate.SportID, seq)
		now := time.Now().UTC()
		f := &CanonicalFixture{
			ID:                  1,
			CanonicalFixtureID:  cid,
			SequenceNumber:      seq,
			SportID:             candidate.SportID,
			SportCode:           canonical.SportIDToCode[candidate.SportID],
			HomeTeam:            candidate.HomeTeam,
			AwayTeam:            candidate.AwayTeam,
			ScheduledStart:      candidate.ScheduledStart.UTC(),
			CompetitionName:     candidate.CompetitionName,
			Gender:              candidate.Gender,
			AgeCategory:         candidate.AgeCategory,
			TeamCategory:        candidate.TeamCategory,
			NaturalIdentityHash: hash,
			IdentityState:       canonical.IdentityMapped,
			VerificationState:   canonical.VerificationVerified,
			FixtureState:        canonical.FixtureLive,
			BettingState:        canonical.BettingOpen,
			SettlementState:     canonical.SettlementPending,
			SettlementCapable:   true,
			ResultResolutionKey: hash,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		a := &FixtureAlias{
			ID:                   1,
			CanonicalFixtureID:   cid,
			Provider:             provider,
			ProviderFixtureID:    providerFixtureID,
			MappingType:          mappingType,
			Source:               source,
			IsActive:             true,
			ResultResolutionPath: hash,
			FirstSeenAt:          now,
			LastSeenAt:           now,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		return f, a, nil
	}

	// 3. Atomic Database Transaction
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, nil, fmt.Errorf("begin jit tx: %w", err)
	}
	defer tx.Rollback()

	// Double check natural key within transaction
	var existingID int64
	var existingCID string
	err = tx.QueryRowContext(ctx, "SELECT id, canonical_fixture_id FROM canonical_fixtures WHERE natural_identity_hash = ? LIMIT 1", hash).Scan(&existingID, &existingCID)
	if err == nil && existingCID != "" {
		_ = tx.Rollback()
		f, fErr := r.GetCanonicalFixture(ctx, existingCID)
		a, aErr := r.AttachAlias(ctx, existingCID, provider, providerFixtureID, mappingType, source)
		if aErr != nil {
			return f, nil, aErr
		}
		return f, a, fErr
	}

	// Increment sequence
	_, err = tx.ExecContext(ctx, "UPDATE canonical_sequence_counter SET current_val = current_val + 1 WHERE id = 1")
	if err != nil {
		return nil, nil, fmt.Errorf("increment sequence: %w", err)
	}
	var nextSeq int64
	err = tx.QueryRowContext(ctx, "SELECT current_val FROM canonical_sequence_counter WHERE id = 1").Scan(&nextSeq)
	if err != nil {
		return nil, nil, fmt.Errorf("query sequence: %w", err)
	}

	cid, err := canonical.FormatCanonicalFixtureID(candidate.SportID, nextSeq)
	if err != nil {
		return nil, nil, fmt.Errorf("format canonical ID: %w", err)
	}

	sportCode := canonical.SportIDToCode[candidate.SportID]
	now := time.Now().UTC()

	// Insert canonical fixture
	insertFixtureQuery := `
		INSERT INTO canonical_fixtures (
			canonical_fixture_id, sequence_number, sport_id, sport_code,
			home_team, away_team, scheduled_start, competition_id, competition_name,
			gender, age_category, team_category,
			natural_identity_hash, identity_state, verification_state, fixture_state,
			betting_state, settlement_state, settlement_capable, result_resolution_key,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)
	`
	res, err := tx.ExecContext(ctx, insertFixtureQuery,
		cid, nextSeq, candidate.SportID, sportCode,
		candidate.HomeTeam, candidate.AwayTeam, candidate.ScheduledStart.UTC(), candidate.CompetitionName,
		candidate.Gender, candidate.AgeCategory, candidate.TeamCategory,
		hash, string(canonical.IdentityMapped), string(canonical.VerificationVerified), string(canonical.FixtureLive),
		string(canonical.BettingOpen), string(canonical.SettlementPending), hash,
		now, now,
	)
	if err != nil {
		_ = tx.Rollback()
		if isUnknownColumnError(err) {
			_ = r.InitSchema(ctx)
			return r.CreateJITCanonicalFixtureTx(ctx, candidate, provider, providerFixtureID, mappingType, source)
		}
		// Handle concurrent race
		if existingRace, rErr := r.GetCanonicalFixtureByNaturalHash(ctx, hash); rErr == nil && existingRace != nil {
			a, aErr := r.AttachAlias(ctx, existingRace.CanonicalFixtureID, provider, providerFixtureID, mappingType, source)
			return existingRace, a, aErr
		}
		return nil, nil, fmt.Errorf("insert jit canonical fixture: %w", err)
	}

	lastID, _ := res.LastInsertId()

	// Insert provider alias atomically in same transaction
	insertAliasQuery := `
		INSERT INTO fixture_aliases (
			canonical_fixture_id, provider, provider_fixture_id,
			mapping_type, source, is_active, result_resolution_path,
			first_seen_at, last_seen_at, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, '{}', ?, ?)
		ON DUPLICATE KEY UPDATE last_seen_at = VALUES(last_seen_at), updated_at = VALUES(updated_at)
	`
	_, err = tx.ExecContext(ctx, insertAliasQuery,
		cid, provider, providerFixtureID,
		string(mappingType), source, hash,
		now, now, now, now,
	)
	if err != nil {
		_ = tx.Rollback()
		if isUnknownColumnError(err) {
			_ = r.InitSchema(ctx)
			return r.CreateJITCanonicalFixtureTx(ctx, candidate, provider, providerFixtureID, mappingType, source)
		}
		return nil, nil, fmt.Errorf("insert jit provider alias: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit jit tx: %w", err)
	}

	f := &CanonicalFixture{
		ID:                  lastID,
		CanonicalFixtureID:  cid,
		SequenceNumber:      nextSeq,
		SportID:             candidate.SportID,
		SportCode:           sportCode,
		HomeTeam:            candidate.HomeTeam,
		AwayTeam:            candidate.AwayTeam,
		ScheduledStart:      candidate.ScheduledStart.UTC(),
		CompetitionName:     candidate.CompetitionName,
		Gender:              candidate.Gender,
		AgeCategory:         candidate.AgeCategory,
		TeamCategory:        candidate.TeamCategory,
		NaturalIdentityHash: hash,
		IdentityState:       canonical.IdentityMapped,
		VerificationState:   canonical.VerificationVerified,
		FixtureState:        canonical.FixtureLive,
		BettingState:        canonical.BettingOpen,
		SettlementState:     canonical.SettlementPending,
		SettlementCapable:   true,
		ResultResolutionKey: hash,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	a := &FixtureAlias{
		CanonicalFixtureID:   cid,
		Provider:             provider,
		ProviderFixtureID:    providerFixtureID,
		MappingType:          mappingType,
		Source:               source,
		IsActive:             true,
		ResultResolutionPath: hash,
		FirstSeenAt:          now,
		LastSeenAt:           now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	return f, a, nil
}

// ResolveCandidate evaluates an incoming candidate fixture across deterministic identity attributes.
// It returns OutcomeExistingMatch, OutcomeConfidentNewMatch, or OutcomeAmbiguous.
func (r *SQLRepository) ResolveCandidate(ctx context.Context, candidate canonical.NaturalCandidate) (ResolutionOutcome, *CanonicalFixture, error) {
	normHome := canonical.NormalizeTeamName(candidate.HomeTeam)
	normAway := canonical.NormalizeTeamName(candidate.AwayTeam)

	// Ambiguity check: blank teams, identical names, or placeholder strings
	if len(normHome) < 2 || len(normAway) < 2 || strings.EqualFold(normHome, normAway) {
		return OutcomeAmbiguous, nil, nil
	}

	// Placeholder tokens
	bannedTokens := map[string]bool{
		"team a": true, "team b": true, "home": true, "away": true,
		"player a": true, "player b": true, "tbd": true, "unknown": true,
	}
	if bannedTokens[normHome] || bannedTokens[normAway] {
		return OutcomeAmbiguous, nil, nil
	}

	candidate.CleanAndClassifyCandidate()
	hash := candidate.NaturalKey()

	// 1. Check if natural candidate matches an existing canonical fixture
	existing, err := r.GetCanonicalFixtureByNaturalHash(ctx, hash)
	if err == nil && existing != nil {
		return OutcomeExistingMatch, existing, nil
	}

	// 2. Legitimate, confident new match
	return OutcomeConfidentNewMatch, nil, nil
}

func (r *SQLRepository) GetAlias(ctx context.Context, provider, providerFixtureID string) (*FixtureAlias, error) {
	if r.db == nil {
		return nil, ErrAliasNotFound
	}
	query := `
		SELECT id, canonical_fixture_id, provider, provider_fixture_id, mapping_type, source,
		       is_active, result_resolution_path,
		       first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at
		FROM fixture_aliases
		WHERE provider = ? AND provider_fixture_id = ?
		LIMIT 1
	`
	var a FixtureAlias
	var mapType, meta string
	err := r.db.QueryRowContext(ctx, query, provider, providerFixtureID).Scan(
		&a.ID, &a.CanonicalFixtureID, &a.Provider, &a.ProviderFixtureID, &mapType, &a.Source,
		&a.IsActive, &a.ResultResolutionPath,
		&a.FirstSeenAt, &a.LastSeenAt, &meta, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAliasNotFound
	}
	if err != nil {
		return nil, err
	}
	a.MappingType = canonical.MappingType(mapType)
	a.MetadataJSON = meta
	return &a, nil
}

func (r *SQLRepository) AttachAlias(ctx context.Context, canonicalID, provider, providerFixtureID string, mappingType canonical.MappingType, source string) (*FixtureAlias, error) {
	if canonicalID == "" || provider == "" || providerFixtureID == "" {
		return nil, errors.New("canonicalID, provider, and providerFixtureID are required")
	}

	// 1. Check existing alias
	existing, err := r.GetAlias(ctx, provider, providerFixtureID)
	if err == nil && existing != nil {
		if existing.CanonicalFixtureID == canonicalID {
			// Idempotent: alias already maps to this canonical fixture; update last_seen_at
			now := time.Now().UTC()
			if r.db != nil {
				_, _ = r.db.ExecContext(ctx, "UPDATE fixture_aliases SET is_active = 1, last_seen_at = ?, updated_at = ? WHERE id = ?", now, now, existing.ID)
			}
			existing.IsActive = true
			existing.LastSeenAt = now
			return existing, nil
		}

		// IDENTITY CONFLICT: Provider ID already mapped to a DIFFERENT canonical fixture!
		conflict := &IdentityConflict{
			ConflictType:        "ALIAS_REASSIGNMENT_ATTEMPT",
			Provider:            provider,
			ProviderFixtureID:   providerFixtureID,
			ExistingCanonicalID: existing.CanonicalFixtureID,
			ProposedCanonicalID: canonicalID,
			EvidenceJSON:        fmt.Sprintf(`{"mapping_type":"%s","source":"%s"}`, mappingType, source),
			Status:              "UNRESOLVED",
			CreatedAt:           time.Now().UTC(),
		}
		_ = r.RecordConflict(ctx, conflict)
		return nil, fmt.Errorf("%w: provider fixture %s:%s is already assigned to %s (attempted to reassign to %s)",
			ErrIdentityConflict, provider, providerFixtureID, existing.CanonicalFixtureID, canonicalID)
	}

	if r.db == nil {
		now := time.Now().UTC()
		return &FixtureAlias{
			ID:                   1,
			CanonicalFixtureID:   canonicalID,
			Provider:             provider,
			ProviderFixtureID:    providerFixtureID,
			MappingType:          mappingType,
			Source:               source,
			IsActive:             true,
			ResultResolutionPath: fmt.Sprintf("%s:%s", provider, providerFixtureID),
			FirstSeenAt:          now,
			LastSeenAt:           now,
			CreatedAt:            now,
			UpdatedAt:            now,
		}, nil
	}

	now := time.Now().UTC()
	resPath := fmt.Sprintf("%s:%s", provider, providerFixtureID)
	query := `
		INSERT INTO fixture_aliases (
			canonical_fixture_id, provider, provider_fixture_id, mapping_type, source,
			is_active, result_resolution_path,
			first_seen_at, last_seen_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, query,
		canonicalID, provider, providerFixtureID, string(mappingType), source,
		resPath,
		now, now, now, now,
	)
	if err != nil {
		if isUnknownColumnError(err) {
			_ = r.InitSchema(ctx)
			return r.AttachAlias(ctx, canonicalID, provider, providerFixtureID, mappingType, source)
		}
		// Check if another worker inserted it concurrently
		if existingConcurrent, cErr := r.GetAlias(ctx, provider, providerFixtureID); cErr == nil && existingConcurrent != nil {
			if existingConcurrent.CanonicalFixtureID == canonicalID {
				return existingConcurrent, nil
			}
			return nil, fmt.Errorf("%w: concurrent alias race on %s:%s", ErrIdentityConflict, provider, providerFixtureID)
		}
		return nil, fmt.Errorf("insert alias: %w", err)
	}

	id, _ := res.LastInsertId()
	return &FixtureAlias{
		ID:                   id,
		CanonicalFixtureID:   canonicalID,
		Provider:             provider,
		ProviderFixtureID:    providerFixtureID,
		MappingType:          mappingType,
		Source:               source,
		IsActive:             true,
		ResultResolutionPath: resPath,
		FirstSeenAt:          now,
		LastSeenAt:           now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

func (r *SQLRepository) GetAliasesForCanonical(ctx context.Context, canonicalID string) ([]FixtureAlias, error) {
	if r.db == nil {
		return nil, nil
	}
	query := `
		SELECT id, canonical_fixture_id, provider, provider_fixture_id, mapping_type, source,
		       is_active, result_resolution_path,
		       first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at
		FROM fixture_aliases
		WHERE canonical_fixture_id = ?
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, canonicalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []FixtureAlias
	for rows.Next() {
		var a FixtureAlias
		var mapType, meta string
		if err := rows.Scan(
			&a.ID, &a.CanonicalFixtureID, &a.Provider, &a.ProviderFixtureID, &mapType, &a.Source,
			&a.IsActive, &a.ResultResolutionPath,
			&a.FirstSeenAt, &a.LastSeenAt, &meta, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		a.MappingType = canonical.MappingType(mapType)
		a.MetadataJSON = meta
		aliases = append(aliases, a)
	}
	return aliases, nil
}

func (r *SQLRepository) RecordConflict(ctx context.Context, conflict *IdentityConflict) error {
	if r.db == nil || conflict == nil {
		return nil
	}
	query := `
		INSERT INTO identity_conflicts (
			conflict_type, provider, provider_fixture_id, existing_canonical_id, proposed_canonical_id,
			evidence, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	evBytes, _ := json.Marshal(conflict.EvidenceJSON)
	_, err := r.db.ExecContext(ctx, query,
		conflict.ConflictType, conflict.Provider, conflict.ProviderFixtureID,
		conflict.ExistingCanonicalID, conflict.ProposedCanonicalID,
		string(evBytes), conflict.Status, conflict.CreatedAt,
	)
	return err
}
