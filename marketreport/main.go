// Command marketreport prints the definitive per-sport market catalogue state:
// every canonical market definition, whether a deterministic settlement rule is
// registered (settleable), and whether CanPlaceMarketCanonical allows placement
// (bettable). It is a read-only dev tool over the live registry; it changes nothing.
//
// Usage: go run ./marketreport   (from the contracts module root)
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

type row struct {
	sportID    int
	sportName  string
	code       string
	name       string
	period     string
	status     string
	settleable bool
	bettable   bool
	reason     string
	timing     string
}

func main() {
	defs := market.ListAllMarketDefinitions()
	rows := make([]row, 0, len(defs))

	for _, d := range defs {
		r := row{
			sportID:   d.SportID,
			sportName: sportName(d.SportID),
			code:      d.MarketCode,
			name:      d.MarketName,
			period:    d.Period,
			status:    d.Status,
		}
		if rule, has := market.Get(d.SportID, d.MarketCode, d.Period, d.RuleVersion); has {
			r.settleable = true
			r.timing = rule.Timing()
		}
		// Placement probe: line-dependent definitions get a representative line so
		// the shape check is honest; specific lines are still validated at runtime.
		var line *float64
		if d.HasLine {
			l := 1.5
			line = &l
		}
		ok, reason := market.CanPlaceMarketCanonical(market.MarketPlacementRequest{
			SportID:             d.SportID,
			CanonicalMarketCode: d.MarketCode,
			Period:              d.Period,
			Line:                line,
			RuleVersion:         d.RuleVersion,
		})
		r.bettable = ok
		r.reason = reason
		rows = append(rows, r)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].sportID != rows[j].sportID {
			return rows[i].sportID < rows[j].sportID
		}
		if rows[i].code != rows[j].code {
			return rows[i].code < rows[j].code
		}
		return rows[i].period < rows[j].period
	})

	var sb strings.Builder
	sb.WriteString("# Market Catalogue — Bettable & Settleable per Sport (generated from contracts/market registry)\n\n")
	sb.WriteString("Bettable = `CanPlaceMarketCanonical` returns true (sport released + ACTIVE definition + settlement rule + certified result domain).\n")
	sb.WriteString("Settleable = deterministic settlement rule registered for the definition's rule version.\n\n")

	// Summary
	type stat struct {
		name                          string
		defs, active, bettable, settle int
	}
	stats := map[int]*stat{}
	for _, r := range rows {
		s := stats[r.sportID]
		if s == nil {
			s = &stat{name: r.sportName}
			stats[r.sportID] = s
		}
		s.defs++
		if r.status == market.MarketStatusActive {
			s.active++
		}
		if r.settleable {
			s.settle++
		}
		if r.bettable {
			s.bettable++
		}
	}
	ids := make([]int, 0, len(stats))
	for id := range stats {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Sport | Definitions | ACTIVE | Bettable | Settleable |\n|---|---:|---:|---:|---:|\n")
	for _, id := range ids {
		s := stats[id]
		sb.WriteString(fmt.Sprintf("| **%s (%d)** | %d | %d | **%d** | **%d** |\n", s.name, id, s.defs, s.active, s.bettable, s.settle))
	}
	sb.WriteString("\n---\n\n")

	// Detail per sport
	last := -1
	for _, r := range rows {
		if r.sportID != last {
			last = r.sportID
			sb.WriteString(fmt.Sprintf("\n## %s (Sport %d)\n\n", r.sportName, r.sportID))
			sb.WriteString("| Canonical Code | Market Name | Period | Def Status | Settleable | Bettable | Placement Reason (if not bettable) |\n|---|---|---|---|---|---|---|\n")
		}
		settleStr := "no"
		if r.settleable {
			settleStr = "yes (rule v registered)"
		}
		betStr := "**yes**"
		if !r.bettable {
			betStr = "no"
		}
		reason := r.reason
		if reason == "" {
			reason = "—"
		}
		if len(reason) > 90 {
			reason = reason[:87] + "..."
		}
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s | %s | %s |\n",
			r.code, r.name, r.period, r.status, settleStr, betStr, strings.ReplaceAll(reason, "|", "/")))
	}

	out := "../Reports/market_catalogue_bettable_settleable.md"
	_ = os.MkdirAll("../Reports", 0o755)
	if err := os.WriteFile(out, []byte(sb.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write error:", err)
	} else {
		fmt.Printf("wrote %s (%d definitions)\n\n", out, len(rows))
	}
	fmt.Print(sb.String())
}

func sportName(id int) string {
	if n, ok := sportNames[id]; ok {
		return n
	}
	return fmt.Sprintf("Sport %d", id)
}
