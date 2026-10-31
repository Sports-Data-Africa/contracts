// Command marketdomains prints canonical_code -> required result domain for every
// catalogue definition. Read-only.
package main

import (
	"fmt"

	"github.com/Sports-Data-Africa/contracts/market"
)

func main() {
	defs := market.ListAllMarketDefinitions()
	for _, d := range defs {
		rule, has := market.Get(d.SportID, d.MarketCode, d.Period, d.RuleVersion)
		domain := "NO_RULE"
		if has {
			domain = rule.RequiredDomain()
		}
		fmt.Printf("%d\t%s\t%s\t%s\n", d.SportID, d.MarketCode, d.Period, domain)
	}
}
