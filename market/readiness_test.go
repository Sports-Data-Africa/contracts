package market

// Tests for the operator release-readiness states used by the all-sports expansion
// programme: MAPPED → INGESTION_VERIFIED → RESULT_DATA_READY → EVALUATOR_READY →
// SHADOW_VERIFIED → RELEASE_APPROVED, tracked per workflow (prematch vs live are
// independent), enforced inside CanPlaceMarketCanonical / CanPlaceMarketCanonicalWorkflow.

import (
	"strings"
	"testing"
)

func placementReq(sportID int, code, period string) MarketPlacementRequest {
	return MarketPlacementRequest{
		SportID:             sportID,
		CanonicalMarketCode: code,
		Period:              period,
		RuleVersion:         ruleVersionFor(sportID, code, period),
	}
}

// TestBlockedDefinitionsDeniedInBothWorkflows proves the three List-2 definitions
// (rugby tries, cricket fours/sixes) can never be placed, in either workflow, until
// their result-data source is demonstrated and the operator removes the denial.
func TestBlockedDefinitionsDeniedInBothWorkflows(t *testing.T) {
	cases := []struct {
		sportID int
		code    string
	}{
		{7, "RB_TOTAL_TRIES"},
		{8, "CK_MOST_FOURS"},
		{8, "CK_MOST_SIXES"},
	}
	for _, c := range cases {
		req := placementReq(c.sportID, c.code, PeriodFullTime)
		req.Line = linePtr(4.5)

		if ok, reason := CanPlaceMarketCanonicalWorkflow(req, WorkflowPrematch); ok {
			t.Fatalf("[%s] must be denied for prematch, got allowed", c.code)
		} else if !strings.Contains(reason, "MARKET_RELEASE_BLOCKED") {
			t.Fatalf("[%s] prematch denial reason = %q, want MARKET_RELEASE_BLOCKED", c.code, reason)
		}

		if ok, reason := CanPlaceMarketCanonicalWorkflow(req, WorkflowLive); ok {
			t.Fatalf("[%s] must be denied for live, got allowed", c.code)
		} else if !strings.Contains(reason, "MARKET_RELEASE_BLOCKED") {
			t.Fatalf("[%s] live denial reason = %q, want MARKET_RELEASE_BLOCKED", c.code, reason)
		}
	}
}

// TestConditionalFootballLiveBlocked proves the corners/cards trio stays prematch
// bettable but is denied in the live workflow until live result evidence lands.
// Prematch vs live readiness must not be inferred from each other.
func TestConditionalFootballLiveBlocked(t *testing.T) {
	cases := []struct {
		code string
	}{
		{"FB_CORNERS_TOTAL"},
		{"FB_CORNERS_1X2"},
		{"FB_CARDS_TOTAL"},
	}
	for _, c := range cases {
		req := MarketPlacementRequest{
			SportID:             1,
			CanonicalMarketCode: c.code,
			Period:              PeriodFullTime,
			RuleVersion:         ruleVersionFor(1, c.code, PeriodFullTime),
			Line:                linePtr(4.5),
		}
		if ok, reason := CanPlaceMarketCanonical(req); !ok {
			t.Fatalf("[%s] prematch placement must remain allowed, reason=%q", c.code, reason)
		}
		if ok, reason := CanPlaceMarketCanonicalWorkflow(req, WorkflowLive); ok {
			t.Fatalf("[%s] live placement must be denied pending live result evidence", c.code)
		} else if !strings.Contains(reason, "MARKET_RELEASE_BLOCKED") {
			t.Fatalf("[%s] live denial reason = %q, want MARKET_RELEASE_BLOCKED", c.code, reason)
		}
	}
}

// TestCoreMarketsUnaffected proves the readiness layer does not regress the
// core football/basketball catalogue (registry-gate behaviour retained).
func TestCoreMarketsUnaffected(t *testing.T) {
	req := MarketPlacementRequest{
		SportID:             1,
		CanonicalMarketCode: "FB_1X2",
		Period:              PeriodFullTime,
		RuleVersion:         ruleVersionFor(1, "FB_1X2", PeriodFullTime),
	}
	if ok, reason := CanPlaceMarketCanonical(req); !ok {
		t.Fatalf("FB_1X2 prematch must remain allowed, reason=%q", reason)
	}
	if ok, reason := CanPlaceMarketCanonicalWorkflow(req, WorkflowLive); !ok {
		t.Fatalf("FB_1X2 live must remain allowed, reason=%q", reason)
	}
}

// TestEffectiveStageProgression verifies first-unmet-stage reporting is honest:
// with no staging evidence yet, the chain legally blocks at INGESTION_VERIFIED —
// a registry rule does NOT prove ingestion or result data — even though the
// evaluator stage is independently achieved.
func TestEffectiveStageProgression(t *testing.T) {
	stage, _ := ReadinessEffectiveStage(1, "FB_1X2", PeriodFullTime, WorkflowPrematch)
	if stage != StageIngestionVerified {
		t.Fatalf("FB_1X2 effective stage = %q, want INGESTION_VERIFIED (mapped+evaluator proven, delivery not yet)", stage)
	}

	// Explicit operator denial does not change the chain stage; it is a release
	// denial recorded separately.
	stageDenied, _ := ReadinessEffectiveStage(7, "RB_TOTAL_TRIES", PeriodFullTime, WorkflowPrematch)
	if stageDenied != StageIngestionVerified {
		t.Fatalf("RB_TOTAL_TRIES effective stage = %q, want INGESTION_VERIFIED (honest chain)", stageDenied)
	}
}

// TestEvidenceGatePolicySwitchesRelease proves the EVIDENCE_GATED policy requires
// the full progression, and that stages move only by explicit evidence flags.
func TestEvidenceGatePolicySwitchesRelease(t *testing.T) {
	t.Setenv("MARKET_READINESS_POLICY", "EVIDENCE_GATED")

	req := MarketPlacementRequest{
		SportID:             1,
		CanonicalMarketCode: "FB_1X2",
		Period:              PeriodFullTime,
		RuleVersion:         ruleVersionFor(1, "FB_1X2", PeriodFullTime),
	}
	if ok, reason := CanPlaceMarketCanonical(req); ok {
		t.Fatal("under EVIDENCE_GATED policy, unverified stages must block release")
	} else if !strings.Contains(reason, "MARKET_RELEASE_NOT_APPROVED") {
		t.Fatalf("reason = %q, want MARKET_RELEASE_NOT_APPROVED", reason)
	}

	// Feed the chain with evidence, stage by stage: nothing skips ahead.
	SetReadinessAchieved(1, "FB_1X2", PeriodFullTime, WorkflowPrematch, StageMapped, true, "mapper branch verified")
	if ok, _ := CanPlaceMarketCanonical(req); ok {
		t.Fatal("MAPPED alone must not approve release")
	}
	SetReadinessAchieved(1, "FB_1X2", PeriodFullTime, WorkflowPrematch, StageIngestionVerified, true, "staging delivery verified")
	SetReadinessAchieved(1, "FB_1X2", PeriodFullTime, WorkflowPrematch, StageResultDataReady, true, "score ingestion verified")
	SetReadinessAchieved(1, "FB_1X2", PeriodFullTime, WorkflowPrematch, StageEvaluatorReady, true, "registry rule v1")
	SetReadinessAchieved(1, "FB_1X2", PeriodFullTime, WorkflowPrematch, StageShadowVerified, true, "shadow parity >=95% over staged fixtures")
	SetReadinessAchieved(1, "FB_1X2", PeriodFullTime, WorkflowPrematch, StageReleaseApproved, true, "operator sign-off")

	if ok, reason := CanPlaceMarketCanonical(req); !ok {
		t.Fatalf("full progression must approve, reason=%q", reason)
	}
}

func linePtr(v float64) *float64 { return &v }
