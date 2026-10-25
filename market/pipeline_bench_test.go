package market_test

import (
	"testing"

	"github.com/Sports-Data-Africa/contracts/market"
)

// BenchmarkPipeline_IngestToNormalize measures raw provider mapping latency
func BenchmarkPipeline_IngestToNormalize(b *testing.B) {
	input := market.ProviderMarketInput{
		Provider:      market.ProviderSportyBet,
		SportID:       1,
		RawMarketID:   "18",
		RawMarketName: "Over/Under",
		RawMarketType: "18",
		RawSpecifier:  "total=2.5",
	}

	b.ReportAllocs()

	for b.Loop() {
		res := market.MapProviderMarket(input)
		if res.CanonicalMarketCode == "" {
			b.Fatal("mapping failed")
		}
	}
}

// BenchmarkPipeline_BetPlacementValidation measures server-side capability & line gating
func BenchmarkPipeline_BetPlacementValidation(b *testing.B) {
	lineVal := 2.5
	req := market.MarketPlacementRequest{
		SportID:             1,
		CanonicalMarketCode: "FB_TOTAL_GOALS",
		Period:              market.PeriodFullTime,
		Line:                &lineVal,
		RuleVersion:         1,
	}

	b.ReportAllocs()

	for b.Loop() {
		allowed, _ := market.CanPlaceMarketCanonical(req)
		if !allowed {
			b.Fatal("placement validation failed")
		}
	}
}

// BenchmarkPipeline_SettlementEvaluation measures deterministic rule evaluation speed
func BenchmarkPipeline_SettlementEvaluation(b *testing.B) {
	lineVal := 2.5
	state := market.CanonicalEventState{
		CanonicalFixtureID: "CF-BENCH-01",
		SportID:            1,
		Status:             market.FixtureStatusFT,
		IsFinished:         true,
		Score:              market.ScoreData{Home: 2, Away: 1},
	}
	bet := market.BetSelection{
		BetID:               "bet-bench-01",
		SportID:             1,
		CanonicalMarketCode: "FB_TOTAL_GOALS",
		Period:              market.PeriodFullTime,
		SelectionCode:       "OVER",
		Line:                &lineVal,
	}

	b.ReportAllocs()

	for b.Loop() {
		res, err := market.Evaluate(state, bet)
		if err != nil || !res.Settled {
			b.Fatal("evaluation failed")
		}
	}
}

// BenchmarkPipeline_SettlementEvaluation_Parallel measures multithreaded evaluation throughput
func BenchmarkPipeline_SettlementEvaluation_Parallel(b *testing.B) {
	lineVal := 2.5
	state := market.CanonicalEventState{
		CanonicalFixtureID: "CF-BENCH-02",
		SportID:            1,
		Status:             market.FixtureStatusFT,
		IsFinished:         true,
		Score:              market.ScoreData{Home: 2, Away: 1},
	}
	bet := market.BetSelection{
		BetID:               "bet-bench-02",
		SportID:             1,
		CanonicalMarketCode: "FB_TOTAL_GOALS",
		Period:              market.PeriodFullTime,
		SelectionCode:       "OVER",
		Line:                &lineVal,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			res, err := market.Evaluate(state, bet)
			if err != nil || !res.Settled {
				b.Fatal("evaluation failed")
			}
		}
	})
}

// BenchmarkPipeline_MultiSportSettlement measures heterogeneous multi-sport evaluation
func BenchmarkPipeline_MultiSportSettlement(b *testing.B) {
	states := []market.CanonicalEventState{
		{
			CanonicalFixtureID: "CF-FB",
			SportID:            1,
			Status:             market.FixtureStatusFT,
			IsFinished:         true,
			Score:              market.ScoreData{Home: 3, Away: 1},
		},
		{
			CanonicalFixtureID: "CF-BB",
			SportID:            2,
			Status:             market.FixtureStatusFT,
			IsFinished:         true,
			Score:              market.ScoreData{Home: 105, Away: 98},
		},
		{
			CanonicalFixtureID: "CF-TT",
			SportID:            13,
			Status:             market.FixtureStatusFT,
			IsFinished:         true,
			Score:              market.ScoreData{Home: 3, Away: 1},
		},
	}

	bets := []market.BetSelection{
		{
			BetID:               "b-1",
			SportID:             1,
			CanonicalMarketCode: "FB_1X2",
			Period:              market.PeriodFullTime,
			SelectionCode:       "1",
		},
		{
			BetID:               "b-2",
			SportID:             2,
			CanonicalMarketCode: "BB_MONEYLINE",
			Period:              market.PeriodFullTime,
			SelectionCode:       "1",
		},
		{
			BetID:               "b-3",
			SportID:             13,
			CanonicalMarketCode: "TT_MATCH_WINNER",
			Period:              market.PeriodFullTime,
			SelectionCode:       "1",
		},
	}

	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		idx := i % 3
		res, err := market.Evaluate(states[idx], bets[idx])
		if err != nil || !res.Settled {
			b.Fatal("evaluation failed")
		}
	}
}
