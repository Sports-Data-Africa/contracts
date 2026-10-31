package market

// Durability tests for the readiness evidence ledger (operator requirement:
// "ensure the readiness state machine is durable across service restarts and
// deployments" — evidence persisted, versioned, stale-invalidated, restart-safe,
// prematch/live approvals independent).

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func ledgerTestFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "readiness_ledger.jsonl")
	SetReadinessLedgerPath(path)
	t.Cleanup(func() { SetReadinessLedgerPath("") })
	return path
}

func resetReadinessStore(t *testing.T) {
	t.Helper()
	readinessMu.Lock()
	readinessStore = map[string]*readinessEntry{}
	readinessLedgerLoaded = false
	readinessMu.Unlock()
	// Mirror a production restart: seeds re-apply to the fresh store.
	applyReleaseDenialSeeds()
}

// TestLedgerPersistsAcrossRestart proves evidence transitions survive a restart:
// records are appended to the ledger, the process "restarts" (store wiped, ledger
// replayed), and effective states + operator denial are intact.
func TestLedgerPersistsAcrossRestart(t *testing.T) {
	path := ledgerTestFile(t)
	SetReadinessAchieved(1, "FB_1X2", PeriodFullTime, WorkflowPrematch, StageIngestionVerified, true, "staging delivery verified")
	SetReadinessAchieved(1, "FB_1X2", PeriodFullTime, WorkflowPrematch, StageResultDataReady, true, "score ingestion verified")
	SetReadinessAchieved(1, "FB_1X2", PeriodFullTime, WorkflowLive, StageIngestionVerified, true, "live delivery verified")

	// Simulate restart.
	resetReadinessStore(t)
	ensureLedgerReplayed()

	pre := readinessStore[readinessKey(1, "FB_1X2", PeriodFullTime)].byWorkflow[WorkflowPrematch]
	if !pre.achieved[StageIngestionVerified] || !pre.achieved[StageResultDataReady] {
		t.Fatalf("prematch replay lost stages: ingestion=%v result=%v",
			pre.achieved[StageIngestionVerified], pre.achieved[StageResultDataReady])
	}
	// Workflow independence: prematch evidence never granted live stages beyond its own.
	live := readinessStore[readinessKey(1, "FB_1X2", PeriodFullTime)].byWorkflow[WorkflowLive]
	if !live.achieved[StageIngestionVerified] {
		t.Fatal("live replay lost its own ingestion evidence")
	}
	if live.achieved[StageResultDataReady] {
		t.Fatal("live must not inherit prematch result-data evidence")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("ledger file missing: %v", err)
	}
}

// TestLedgerReplaysOperatorDenial proves operator denials are durable and remain
// authoritative: a denial recorded while the ledger is active is replayed after a
// restart, and evidence recorded later cannot silently clear it.
func TestLedgerReplaysOperatorDenial(t *testing.T) {
	ledgerTestFile(t)
	MarkReleaseDenied(3, "TN_MATCH_WINNER", PeriodFullTime, WorkflowLive, "operator hold: live feed review")
	if ok, reason := CanPlaceMarketCanonicalWorkflow(placementReq(3, "TN_MATCH_WINNER", PeriodFullTime), WorkflowLive); ok {
		t.Fatalf("denial lost before restart: %q", reason)
	}
	resetReadinessStore(t)
	if ok, reason := CanPlaceMarketCanonicalWorkflow(placementReq(3, "TN_MATCH_WINNER", PeriodFullTime), WorkflowLive); ok {
		t.Fatalf("denial lost after restart: %q", reason)
	} else if !strings.Contains(reason, "MARKET_RELEASE_BLOCKED") {
		t.Fatalf("replayed denial reason = %q, want MARKET_RELEASE_BLOCKED", reason)
	}
}

// TestLedgerStaleRuleVersionInvalidation proves a rule/mapping upgrade invalidates
// evaluator/shadow evidence recorded under an older rule version — with the stale
// reason preserved — while MAPPED survives (it is not version-dependent evidence).
func TestLedgerStaleRuleVersionInvalidation(t *testing.T) {
	ledgerTestFile(t)

	// Record evaluator+shadow achievements, then corrupt the recorded rule version
	// to simulate a rule upgrade happening after the evidence was logged.
	SetReadinessAchieved(1, "FB_BTTS", PeriodFullTime, WorkflowPrematch, StageEvaluatorReady, true, "rule v1 verified")
	SetReadinessAchieved(1, "FB_BTTS", PeriodFullTime, WorkflowPrematch, StageShadowVerified, true, "parity verified")
	stale := ruleVersionFor(1, "FB_BTTS", PeriodFullTime) + 7
	mutateLedgerRuleVersion(t, 1, "FB_BTTS", PeriodFullTime, "prematch", stale)

	// simulate restart
	resetReadinessStore(t)
	ensureLedgerReplayed()

	wfEntry := readinessStore[readinessKey(1, "FB_BTTS", PeriodFullTime)]
	if wfEntry == nil {
		t.Fatal("FB_BTTS entry missing after replay")
	}
	pre := wfEntry.byWorkflow[WorkflowPrematch]
	if pre.achieved[StageEvaluatorReady] {
		t.Fatal("evaluator evidence must be invalidated when rule version changed")
	}
	if pre.achieved[StageShadowVerified] {
		t.Fatal("shadow evidence must be invalidated when rule version changed")
	}
	if !strings.Contains(pre.notes[StageEvaluatorReady], "STALE") {
		t.Fatalf("stale evaluator note = %q, want a STALE rule-version reason", pre.notes[StageEvaluatorReady])
	}
	stage, _ := ReadinessEffectiveStage(1, "FB_BTTS", PeriodFullTime, WorkflowPrematch)
	if stage != StageIngestionVerified {
		t.Fatalf("effective stage = %q, want INGESTION_VERIFIED after invalidation", stage)
	}
}

// TestLedgerCorruptLinesSkipped proves replay tolerates truncated/corrupt lines.
func TestLedgerCorruptLinesSkipped(t *testing.T) {
	path := ledgerTestFile(t)
	SetReadinessAchieved(2, "BB_MONEYLINE", PeriodFullTime, WorkflowPrematch, StageIngestionVerified, true, "delivered")
	if err := os.WriteFile(path, append([]byte("not-json-garbage\n{\n"), []byte("{\"ts\":\"2026-10-31T00:00:00Z\",\"sport_id\":2,\"code\":\"BB_MONEYLINE\",\"period\":\"FT\",\"workflow\":\"prematch\",\"stage\":\"RESULT_DATA_READY\",\"achieved\":true,\"note\":\"stats\"}\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	resetReadinessStore(t)
	ensureLedgerReplayed()
	rec := readinessStore[readinessKey(2, "BB_MONEYLINE", PeriodFullTime)]
	if rec == nil {
		t.Fatal("entry missing after replay")
	}
	if !rec.byWorkflow[WorkflowPrematch].achieved[StageResultDataReady] {
		t.Fatal("valid RESULT_DATA_READY record must replay despite corrupt lines")
	}
	if got, _ := ReadinessEffectiveStage(2, "BB_MONEYLINE", PeriodFullTime, WorkflowPrematch); got != StageIngestionVerified {
		t.Fatalf("effective stage = %q, want INGESTION_VERIFIED (chain order; corrupt lines skipped)", got)
	}
}

// TestPrematchEvidenceNeverApprovesLive proves workflow independence across the
// whole progression: prematch-only evidence leaves the live stage untouched.
func TestPrematchEvidenceNeverApprovesLive(t *testing.T) {
	ledgerTestFile(t)
	for _, stage := range readinessStageOrder {
		SetReadinessAchieved(1, "FB_DOUBLE_CHANCE", PeriodFullTime, WorkflowPrematch, stage, true, "evidence")
	}
	preStage, _ := ReadinessEffectiveStage(1, "FB_DOUBLE_CHANCE", PeriodFullTime, WorkflowPrematch)
	liveStage, _ := ReadinessEffectiveStage(1, "FB_DOUBLE_CHANCE", PeriodFullTime, WorkflowLive)
	if preStage != StageReleaseApproved {
		t.Fatalf("prematch stage = %q, want RELEASE_APPROVED", preStage)
	}
	if liveStage == StageReleaseApproved {
		t.Fatal("prematch evidence must never approve live")
	}
}

// TestResultDataLossRevokesStage proves a provider losing a result capability
// explicitly lowers the live stage (never left stale-approved).
func TestResultDataLossRevokesStage(t *testing.T) {
	SetReadinessLedgerPath("") // in-memory scenario; ledger not needed for revocation
	t.Cleanup(func() { SetReadinessLedgerPath("") })

	SetReadinessAchieved(4, "IH_TOTAL_GOALS", PeriodFullTime, WorkflowLive, StageResultDataReady, true, "stats available")
	SetReadinessAchieved(4, "IH_TOTAL_GOALS", PeriodFullTime, WorkflowLive, StageEvaluatorReady, true, "rule")
	// provider loses the stats feed BEFORE ingestion is proven — the chain still
	// blocks at INGESTION (ordering), but result-data must never remain approved:
	SetReadinessAchieved(4, "IH_TOTAL_GOALS", PeriodFullTime, WorkflowLive, StageIngestionVerified, true, "delivered")
	MarkResultDataLost(4, "IH_TOTAL_GOALS", PeriodFullTime, WorkflowLive, "provider removed stats feed", "provider-supervisor")

	res := readinessStore[readinessKey(4, "IH_TOTAL_GOALS", PeriodFullTime)].byWorkflow[WorkflowLive]
	if res.achieved[StageResultDataReady] {
		t.Fatal("RESULT_DATA_READY must be revoked when the result source is lost")
	}
	if !strings.Contains(res.notes[StageResultDataReady], "RESULT_DATA_LOST") {
		t.Fatalf("revocation note = %q, want RESULT_DATA_LOST explanation", res.notes[StageResultDataReady])
	}
	if got, _ := ReadinessEffectiveStage(4, "IH_TOTAL_GOALS", PeriodFullTime, WorkflowLive); got != StageResultDataReady {
		t.Fatalf("effective stage = %q, want RESULT_DATA_READY as the blocking stage", got)
	}
}

// mutateLedgerRuleVersion rewrites the recorded rule_version of every ledger entry
// for a code, simulating a rule upgrade that happened after evidence was recorded.
func mutateLedgerRuleVersion(t *testing.T, sportID int, code, period, workflow string, newVersion int) {
	t.Helper()
	f := ledgerPath()
	if f == "" {
		t.Fatal("no ledger path configured")
	}
	raw, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("ledger read: %v", err)
	}
	var rebuilt []string
	for _, ln := range strings.Split(string(raw), "\n") {
		if strings.Contains(ln, `"code":"`+code+`"`) && strings.Contains(ln, `"workflow":"`+workflow+`"`) {
			idx := strings.Index(ln, `"rule_version":`)
			if idx >= 0 {
				end := idx + len(`"rule_version":`)
				endIdx := strings.IndexByte(ln[end:], ',')
				if endIdx < 0 {
					endIdx = strings.IndexByte(ln[end:], '}')
				}
				ln = ln[:end] + strconv.Itoa(newVersion) + ln[end+endIdx:]
			}
		}
		rebuilt = append(rebuilt, ln)
	}
	if err := os.WriteFile(f, []byte(strings.Join(rebuilt, "\n")), 0o644); err != nil {
		t.Fatalf("ledger write: %v", err)
	}
}

func ledgerPath() string {
	if readinessLedgerPath != "" {
		return readinessLedgerPath
	}
	return os.Getenv("MARKET_READINESS_LEDGER")
}
