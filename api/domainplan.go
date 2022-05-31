package api

import (
	"encoding/json"
)

//go:generate stringer -type=DomainPlan -linecomment
type DomainPlan int

const (
	DomainPlanFree     DomainPlan = iota // free
	DomainPlanBasic                      // basic
	DomainPlanAdvanced                   // advanced
)

func (p DomainPlan) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *DomainPlan) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	*p = DomainPlan(0)

	for _, x := range []DomainPlan{DomainPlanFree, DomainPlanBasic, DomainPlanAdvanced} {
		if x.String() == s {
			*p = x
			return nil
		}
	}
	return nil
}
