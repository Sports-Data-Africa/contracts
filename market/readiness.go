package market

// Operator-agreed release readiness states for the all-sports expansion programme.
//
// Progression (operator mandate): MAPPED → INGESTION_VERIFIED → RESULT_DATA_READY →
// EVALUATOR_READY → SHADOW_VERIFIED → RELEASE_APPROVED, tracked independently for
// prematch and live. A definition is only releasable in a workflow when every prior
// stage is achieved in that workflow, OR when the readiness policy is LEGACY_REGISTRY
// (the current default), in which case only explicit operator denial is enforced —
// preserving today's catalogue behaviour until staging evidence switches the policy.
//
// This file is the single source of truth for release decisions:
// CanPlaceMarketCanonical resolves with the prematch workflow; callers that know
// they are operating the live phase must use CanPlaceMarketCanonicalWorkflow.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Workflow identifies the betting phase readiness is tracked for.
type Workflow string

const (
	WorkflowPrematch Workflow = "prematch"
	WorkflowLive     Workflow = "live"
)

// ReadinessStage is one step of the operator progression.
type ReadinessStage string

const (
	StageMapped            ReadinessStage = "MAPPED"
	StageIngestionVerified ReadinessStage = "INGESTION_VERIFIED"
	StageResultDataReady   ReadinessStage = "RESULT_DATA_READY"
	StageEvaluatorReady    ReadinessStage = "EVALUATOR_READY"
	StageShadowVerified    ReadinessStage = "SHADOW_VERIFIED"
	StageReleaseApproved   ReadinessStage = "RELEASE_APPROVED"
)

var readinessStageOrder = []ReadinessStage{
	StageMapped, StageIngestionVerified, StageResultDataReady,
	StageEvaluatorReady, StageShadowVerified, StageReleaseApproved,
}

type readinessFlags struct {
	achieved map[ReadinessStage]bool
	notes    map[ReadinessStage]string
	// denial is an explicit operator decision that the definition must not be
	// released in this workflow regardless of stage achievements. Empty string
	// means not denied. It only ever changes via MarkReleaseDenied / an explicit
	// RELEASE_APPROVED achievement, never implicitly.
	denial string
}

type readinessEntry struct {
	sportID int
	code    string
	period  string
	byWorkflow map[Workflow]*readinessFlags
}

var readinessMu sync.RWMutex
var readinessStore = map[string]*readinessEntry{}

func readinessKey(sportID int, code, period string) string {
	return fmt.Sprintf("%d:%s:%s", sportID, strings.ToUpper(strings.TrimSpace(code)), strings.ToUpper(strings.TrimSpace(period)))
}

func ensureReadiness(sportID int, code, period string, wf Workflow) *readinessFlags {
	key := readinessKey(sportID, code, period)
	e, ok := readinessStore[key]
	if !ok {
		e = &readinessEntry{sportID: sportID, code: strings.ToUpper(code), period: strings.ToUpper(period), byWorkflow: map[Workflow]*readinessFlags{}}
		readinessStore[key] = e
	}
	wfKey := wf
	if wf != WorkflowLive {
		wfKey = WorkflowPrematch
	}
	flags, ok := e.byWorkflow[wfKey]
	if !ok {
		flags = &readinessFlags{achieved: map[ReadinessStage]bool{}, notes: map[ReadinessStage]string{}}
		// Lazily derive the two stages the registry itself can prove.
		if _, hasDef := GetMarketDefinition(sportID, code, period); hasDef {
			flags.achieved[StageMapped] = true
		}
		if ruleVersion := ruleVersionFor(sportID, code, period); ruleVersion > 0 {
			if _, hasRule := Get(sportID, code, period, ruleVersion); hasRule {
				flags.achieved[StageEvaluatorReady] = true
			}
		}
		e.byWorkflow[wfKey] = flags
	}
	return flags
}

// SetReadinessAchieved records (or revokes) a stage achievement with an evidence note.
// Pipeline tools call this as staging evidence accumulates; it is never inferred from
// registry existence alone. Explicitly achieving RELEASE_APPROVED clears an operator
// denial (an approval supersedes a stale block).
//
// When MARKET_READINESS_LEDGER is configured the transition is persisted, with the
// current rule version stamped on evaluator/shadow achievements so a later rule or
// mapping upgrade can invalidate stale downstream readiness automatically.
func SetReadinessAchieved(sportID int, code, period string, wf Workflow, stage ReadinessStage, achieved bool, note string) {
	ensureLedgerReplayed()

	readinessMu.Lock()
	flags := ensureReadiness(sportID, code, period, wf)
	flags.achieved[stage] = achieved
	if achieved && stage == StageReleaseApproved {
		flags.denial = ""
	}
	if note == "" {
		delete(flags.notes, stage)
	} else {
		flags.notes[stage] = note
	}
	readinessMu.Unlock()

	rec := evidenceRecord{
		SportID: sportID, Code: strings.ToUpper(code), Period: strings.ToUpper(period),
		Workflow: string(wf), Stage: string(stage), Achieved: achieved, Note: note,
		Source: "readiness.go",
	}
	if stage == StageEvaluatorReady || stage == StageShadowVerified {
		rec.RuleVersion = ruleVersionFor(sportID, code, period)
	}
	appendReadinessLedger(rec)
}

// ReadinessEffectiveStage returns the first unmet stage in the operator progression,
// i.e. the blocking stage, for the workflow. RELEASE_APPROVED means fully releasable.
func ReadinessEffectiveStage(sportID int, code, period string, wf Workflow) (ReadinessStage, string) {
	ensureLedgerReplayed()
	readinessMu.RLock()
	flags := ensureReadiness(sportID, code, period, wf)
	readinessMu.RUnlock()
	for _, stage := range readinessStageOrder {
		if !flags.achieved[stage] {
			return stage, flags.notes[stage]
		}
	}
	return StageReleaseApproved, ""
}

// ReadinessPolicy selects how release decisions treat unachieved stages.
//   - LEGACY_REGISTRY (default): today's behaviour — registry-active definitions with
//     rules are releasable, except where the operator has explicitly denied release.
//   - EVIDENCE_GATED: the full operator progression must be achieved in the workflow.
func ReadinessPolicy() string {
	p := strings.ToUpper(strings.TrimSpace(os.Getenv("MARKET_READINESS_POLICY")))
	if p == "EVIDENCE_GATED" {
		return p
	}
	return "LEGACY_REGISTRY"
}

func releaseDecision(sportID int, code, period string, wf Workflow) (bool, string) {
	ensureLedgerReplayed()

	readinessMu.RLock()
	flags := ensureReadiness(sportID, code, period, wf)
	denial := flags.denial
	readinessMu.RUnlock()

	// Explicit operator denial dominates every stage achievement.
	if denial != "" {
		return false, "MARKET_RELEASE_BLOCKED: " + denial
	}

	if ReadinessPolicy() == "EVIDENCE_GATED" {
		stage, note := ReadinessEffectiveStage(sportID, code, period, wf)
		if stage != StageReleaseApproved {
			if note != "" {
				return false, fmt.Sprintf("MARKET_RELEASE_NOT_APPROVED: blocking stage %s: %s", stage, note)
			}
			return false, fmt.Sprintf("MARKET_RELEASE_NOT_APPROVED: blocking stage %s (workflow %s)", stage, wf)
		}
	}
	return true, ""
}

// MarkReleaseDenied records an explicit operator decision that a definition must not
// be released in the given workflow (list from Reports/market_release_readiness.md).
func MarkReleaseDenied(sportID int, code, period string, wf Workflow, reason string) {
	ensureLedgerReplayed()
	readinessMu.Lock()
	defer readinessMu.Unlock()
	flags := ensureReadiness(sportID, code, period, wf)
	flags.achieved[StageReleaseApproved] = false
	flags.denial = reason
	appendReadinessLedger(evidenceRecord{
		Timestamp: time.Now().UTC(), SportID: sportID, Code: strings.ToUpper(code),
		Period: strings.ToUpper(period), Workflow: string(wf),
		Stage: string(StageReleaseApproved), Achieved: false,
		Note: "OPERATOR_DENIAL: " + reason, Source: "readiness.go",
	})
}

// resultVersionLoss is called when a result source loses availability so that
// RESULT_DATA_READY is revoked explicitly (never left stale-approved) and the
// reason is recorded. Under EVIDENCE_GATED this lowers the effective stage; under
// LEGACY_REGISTRY it is recorded for diagnostics and audit.
func MarkResultDataLost(sportID int, code, period string, wf Workflow, reason, source string) {
	SetReadinessAchieved(sportID, code, period, wf, StageResultDataReady, false,
		"RESULT_DATA_LOST: "+reason+" (source: "+source+")")
}

// ---------------- durable evidence ledger ----------------
//
// The readiness store is authoritative only if evidence survives service restarts
// and deployments. When MARKET_READINESS_LEDGER is configured (an append-only
// JSON Lines file on shared storage), every stage transition is persisted with a
// timestamp, source and rule version, and replayed on process start. Corrupt lines
// are skipped, never fatal. Deployment: mount the same file for every service that
// resolves release decisions.

type evidenceRecord struct {
	Timestamp   time.Time `json:"ts"`
	SportID     int       `json:"sport_id"`
	Code        string    `json:"code"`
	Period      string    `json:"period"`
	Workflow    string    `json:"workflow"`
	Stage       string    `json:"stage"`
	Achieved    bool      `json:"achieved"`
	Note        string    `json:"note,omitempty"`
	RuleVersion int       `json:"rule_version,omitempty"`
	Source      string    `json:"source,omitempty"`
}

var readinessLedgerPath string
var readinessLedgerLoaded bool

// SetReadinessLedgerPath configures the durable evidence file. Safe to call before
// any readiness mutation; call again with "" to disable persistence (tests).
func SetReadinessLedgerPath(path string) {
	readinessMu.Lock()
	readinessLedgerPath = path
	readinessLedgerLoaded = path == ""
	readinessMu.Unlock()
}

func ledgerPathConfigured() string {
	if readinessLedgerPath != "" {
		return readinessLedgerPath
	}
	return strings.TrimSpace(os.Getenv("MARKET_READINESS_LEDGER"))
}

func ensureLedgerReplayed() {
	path := ledgerPathConfigured()
	if path == "" || readinessLedgerLoaded {
		return
	}
	readinessMu.Lock()
	defer readinessMu.Unlock()
	if readinessLedgerLoaded {
		return
	}
	readinessLedgerLoaded = true

	// NOTE: caller holds readinessMu for the whole replay; the record appliers below
	// must therefore never take the lock themselves.

	f, err := os.Open(path)
	if err != nil {
		return // no ledger yet; first write creates it
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue // tolerate corrupt/truncated partial lines
		}
		var rec evidenceRecord
		if json.Unmarshal([]byte(line), &rec) != nil {
			continue
		}
		replayLedgerRecord(rec)
	}
}

func replayLedgerRecord(rec evidenceRecord) {
	// Precondition: readinessMu is already held by the caller (ensureLedgerReplayed).
	stage := ReadinessStage(rec.Stage)
	var wf Workflow
	if strings.EqualFold(rec.Workflow, string(WorkflowLive)) {
		wf = WorkflowLive
	} else {
		wf = WorkflowPrematch
	}

	// Stale-evidence invalidation: evaluator/shadow achievements recorded against
	// an older rule version must not survive a rule/mapping upgrade.
	if stage == StageEvaluatorReady || stage == StageShadowVerified {
		if rec.RuleVersion > 0 && rec.RuleVersion != ruleVersionFor(rec.SportID, rec.Code, rec.Period) {
			flags := ensureReadiness(rec.SportID, rec.Code, rec.Period, wf)
			flags.achieved[stage] = false
			flags.notes[stage] = fmt.Sprintf("STALE: rule version changed %d -> %d since evidence was recorded",
				rec.RuleVersion, ruleVersionFor(rec.SportID, rec.Code, rec.Period))
			return
		}
	}

	flags := ensureReadiness(rec.SportID, rec.Code, rec.Period, wf)
	if rec.Stage == string(StageReleaseApproved) && !rec.Achieved {
		// Operator denial replay: authoritative, evidence cannot clear it silently.
		flags.achieved[StageReleaseApproved] = false
		flags.denial = rec.Note
		return
	}
	flags.achieved[stage] = rec.Achieved
	if rec.Achieved && stage == StageReleaseApproved {
		flags.denial = ""
	}
	if rec.Note == "" {
		delete(flags.notes, stage)
	} else {
		flags.notes[stage] = rec.Note
	}
}

func appendReadinessLedger(rec evidenceRecord) {
	path := ledgerPathConfigured()
	if path == "" {
		return
	}
	rec.Timestamp = time.Now().UTC()
	if f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		defer f.Close()
		if b, mErr := json.Marshal(rec); mErr == nil {
			_, _ = f.Write(append(b, '\n'))
		}
	}
}

// blockedEverywhere: no provider mapping path / no evidenced result source.
// liveBlockedOnly: result-data evidence pending for the live workflow.
var releaseDenialSeeds = []struct {
	sportID   int
	code      string
	period    string
	liveOnly  bool
	reason    string
}{
	{7, "RB_TOTAL_TRIES", PeriodFullTime, false,
		"mapRugby deliberately excludes try/tries markets (provider_mapping.go) and tries result ingestion is not evidenced"},
	{8, "CK_MOST_FOURS", PeriodFullTime, false,
		"fours aggregates result source is not evidenced in the pipeline"},
	{8, "CK_MOST_SIXES", PeriodFullTime, false,
		"sixes aggregates result source is not evidenced in the pipeline"},
	{1, "FB_CORNERS_TOTAL", PeriodFullTime, true,
		"corners counts proven for live settlement is pending (operator staging gate)"},
	{1, "FB_CORNERS_1X2", PeriodFullTime, true,
		"corners counts proven for live settlement is pending (operator staging gate)"},
	{1, "FB_CORNERS_ODD_EVEN", PeriodFullTime, true,
		"corners counts proven for live settlement is pending (operator staging gate)"},
	{1, "FB_HT_CORNERS_TOTAL", PeriodHalfTime, true,
		"corners counts proven for live settlement is pending (operator staging gate)"},
	{1, "FB_CARDS_TOTAL", PeriodFullTime, true,
		"cards counts proven for live settlement is pending (operator staging gate)"},
	{1, "FB_CARDS_1X2", PeriodFullTime, true,
		"cards counts proven for live settlement is pending (operator staging gate)"},
	{1, "FB_CARDS_ODD_EVEN", PeriodFullTime, true,
		"cards counts proven for live settlement is pending (operator staging gate)"},
}

func init() {
	applyReleaseDenialSeeds()
}

// applyReleaseDenialSeeds mirrors a production restart: the operator denial list is
// re-applied to every fresh process (and after test-store wipes).
func applyReleaseDenialSeeds() {
	for _, s := range releaseDenialSeeds {
		if s.liveOnly {
			MarkReleaseDenied(s.sportID, s.code, s.period, WorkflowLive, s.reason)
			continue
		}
		MarkReleaseDenied(s.sportID, s.code, s.period, WorkflowPrematch, s.reason)
		MarkReleaseDenied(s.sportID, s.code, s.period, WorkflowLive, s.reason)
	}
}
func ruleVersionFor(sportID int, code, period string) int {
	def, ok := GetMarketDefinition(sportID, code, period)
	if !ok {
		return 0
	}
	if def.RuleVersion <= 0 {
		return 1
	}
	return def.RuleVersion
}
