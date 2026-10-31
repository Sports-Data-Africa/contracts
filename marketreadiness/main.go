// Command marketreadiness classifies every canonical market definition against the
// release criteria and prints the two release lists (READY / NOT READY) with reasons.
//
// Classification (code-derived):
//   READY       — ACTIVE definition + settlement rule + certified domain, and the
//                 required result domain is match-score-like data that prematch and
//                 live workflows both carry (score, period score, points, sets,
//                 games, rounds, innings, penalties, extra time).
//   CONDITIONAL — ACTIVE + rule, but the result domain is corners/cards counts:
//                 bettable wherever the provider supplies those markets, settleable
//                 only where corners/cards counts are proven to reach the result
//                 pipeline for that workflow.
//   NOT_READY   — requires DOMAIN_TEAM_STATS (fours/sixes/tries aggregates): no
//                 evidenced result-data path, and one code also lacks any provider
//                 mapping path entirely.
//
// Usage: go run ./marketreadiness   (from the contracts module root)
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Sports-Data-Africa/contracts/market"
)

var sportNames = map[int]string{
	1: "Football", 2: "Basketball", 3: "Tennis", 4: "Ice Hockey", 5: "Volleyball",
	6: "MMA", 7: "Rugby", 8: "Cricket", 9: "American Football", 10: "Handball",
	11: "Baseball", 13: "Table Tennis",
}

// scoreLikeDomains are carried by match state that both prematch and live feeds and
// the result pipeline deliver (scores, period scores, racquet game/set/point counts,
// rounds, innings overs). Everything else needs dedicated evidence.
var scoreLikeDomains = map[string]bool{
	market.DomainScore:        true,
	market.DomainPeriodScore:  true,
	market.DomainPoints:       true,
	market.DomainSets:         true,
	market.DomainGames:        true,
	market.DomainRounds:       true,
	market.DomainInnings:      true,
	market.DomainOvers:        true,
	market.DomainExtraTime:    true,
	market.DomainPenalties:    true,
}

// noProviderMappingPath are catalogue codes that no provider mapper ever returns
// (static analysis of provider_mapping.go: mapSportyBet / mapOdibets returns).
var noProviderMappingPath = map[string]bool{
	"RB_TOTAL_TRIES": true,
}

type item struct {
	sportID   int
	sportName string
	code      string
	period    string
	status    string
	list      string // READY / CONDITIONAL / NOT_READY
	reason    string
}

func main() {
	defs := market.ListAllMarketDefinitions()
	items := make([]item, 0, len(defs))

	for _, d := range defs {
		r := item{
			sportID:   d.SportID,
			sportName: sportName(d.SportID),
			code:      d.MarketCode,
			period:    d.Period,
			status:    d.Status,
		}
		if d.Status != market.MarketStatusActive {
			r.list = "NOT_READY"
			r.reason = "Catalogue definition is " + d.Status
			items = append(items, r)
			continue
		}
		rule, has := market.Get(d.SportID, d.MarketCode, d.Period, d.RuleVersion)
		if !has {
			r.list = "NOT_READY"
			r.reason = "No settlement rule registered (v" + fmt.Sprint(d.RuleVersion) + ")"
			items = append(items, r)
			continue
		}
		domain := rule.RequiredDomain()
		switch {
		case noProviderMappingPath[d.MarketCode]:
			r.list = "NOT_READY"
			r.reason = "No provider mapping path (mapSportyBet/mapOdibets never emit this code); needs domain " + domain
		case domain == market.DomainTeamStats:
			r.list = "NOT_READY"
			r.reason = "Requires " + domain + " (aggregate team stat counts); result-data delivery is not evidenced in the pipeline"
		case domain == market.DomainCorners || domain == market.DomainCards:
			r.list = "CONDITIONAL"
			r.reason = "Bettable prematch; settlement requires " + domain + " counts proven to reach the result pipeline for the workflow"
		case scoreLikeDomains[domain]:
			r.list = "READY"
			r.reason = "Score-domain market: settled from match state both workflows carry"
		default:
			r.list = "CONDITIONAL"
			r.reason = "Unclassified domain " + domain + " — needs manual evidence review"
		}
		items = append(items, r)
	}

	sort.Slice(items, func(i, j int) bool {
		rank := map[string]int{"READY": 0, "CONDITIONAL": 1, "NOT_READY": 2}
		if rank[items[i].list] != rank[items[j].list] {
			return rank[items[i].list] < rank[items[j].list]
		}
		if items[i].sportID != items[j].sportID {
			return items[i].sportID < items[j].sportID
		}
		if items[i].code != items[j].code {
			return items[i].code < items[j].code
		}
		return items[i].period < items[j].period
	})

	var sb strings.Builder
	sb.WriteString("# Market Release Readiness — canonical registry vs end-to-end evidence\n\n")
	sb.WriteString("Generated from the live registry (contracts/market). Release criteria: the workflow must be able to DELIVER (provider), MAP, PROJECT, EXPOSE, ACCEPT and SETTLE the market with deterministic result data.\n\n")
	sb.WriteString("**Boundary**: provider delivery per fixture is runtime evidence (staging `exposure_diagnostics`, settlement result rows). This report fixes the static chain; runtime collection remains required before going live.\n\n")

	counts := map[string]int{}
	for _, r := range items {
		counts[r.list]++
	}
	sb.WriteString(fmt.Sprintf("**READY: %d · CONDITIONAL: %d · NOT READY: %d · total: %d**\n\n", counts["READY"], counts["CONDITIONAL"], counts["NOT_READY"], len(items)))

	last := ""
	for _, r := range items {
		if r.list != last {
			last = r.list
			sb.WriteString(fmt.Sprintf("\n## %s\n\n", map[string]string{
				"READY":       "### List 1 — Markets ready for release",
				"CONDITIONAL": "### List 1b — Prematch-ready, live-release conditional on result-data evidence",
				"NOT_READY":   "### List 2 — Not ready for release",
			}[r.list]))
			sb.WriteString("| Sport | Code | Period | Status | Reason / remaining work |\n|---|---|---|---|---|\n")
		}
		reason := r.reason
		if len(reason) > 120 {
			reason = reason[:117] + "..."
		}
		sb.WriteString(fmt.Sprintf("| %s (%d) | `%s` | %s | %s | %s |\n", r.sportName, r.sportID, r.code, r.period, r.status, strings.ReplaceAll(reason, "|", "/")))
	}

	out := "../Reports/market_release_readiness.md"
	_ = os.MkdirAll("../Reports", 0o755)
	if err := os.WriteFile(out, []byte(sb.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write error:", err)
	} else {
		fmt.Fprintf(os.Stderr, "wrote %s\n", out)
	}

	// Approvals appendix — the exact release lists requested by the operator:
	//   approvedPrematch = READY + CONDITIONAL (prematch exposure allowed; live blocked
	//     for CONDITIONAL until result-data evidence)
	//   approvedLive     = READY only
	//   blockedEverywhere= NOT_READY
	approvedPrematch := make([]item, 0, len(items))
	approvedLive := make([]item, 0, len(items))
	blocked := make([]item, 0, len(items))
	for _, r := range items {
		switch r.list {
		case "READY":
			approvedPrematch = append(approvedPrematch, r)
			approvedLive = append(approvedLive, r)
		case "CONDITIONAL":
			approvedPrematch = append(approvedPrematch, r)
		case "NOT_READY":
			blocked = append(blocked, r)
		}
	}
	var ap strings.Builder
	ap.WriteString("\n---\n\n")
	ap.WriteString("## Operator approval lists (exact, code-level)\n\n")
	ap.WriteString(fmt.Sprintf("**Approved for prematch exposure (staging verification group): %d markets**\n\n", len(approvedPrematch)))
	for _, r := range approvedPrematch {
		ap.WriteString(fmt.Sprintf("- [%d %s] `%s` (%s)\n", r.sportID, r.sportName, r.code, r.period))
	}
	ap.WriteString(fmt.Sprintf("\n**Approved for live exposure: %d markets**\n\n", len(approvedLive)))
	for _, r := range approvedLive {
		ap.WriteString(fmt.Sprintf("- [%d %s] `%s` (%s)\n", r.sportID, r.sportName, r.code, r.period))
	}
	ap.WriteString(fmt.Sprintf("\n**Blocked in both workflows: %d markets**\n\n", len(blocked)))
	for _, r := range blocked {
		ap.WriteString(fmt.Sprintf("- [%d %s] `%s` (%s) — %s\n", r.sportID, r.sportName, r.code, r.period, r.reason))
	}
	ap.WriteString("\n**Conditions attached:** staging evidence (delivery → mapping → projection → exposure → acceptance → settlement) must be captured per market before production release; the CONDITIONAL three stay live-blocked until their corners/cards result data is demonstrated; SETTLEMENT_MODE ACTIVE requires SHADOW parity sign-off.\n")
	_ = os.WriteFile(out, []byte(sb.String()+ap.String()), 0o644)
	fmt.Print(ap.String())
}

func sportName(id int) string {
	if n, ok := sportNames[id]; ok {
		return n
	}
	return fmt.Sprintf("Sport %d", id)
}
