package api

import (
	"encoding/json"
)

//go:generate stringer -type=DomainType -linecomment
type DomainType int

const (
	_                    DomainType = iota
	DomainTypeCloudflare            // cloudflare
	DomainTypeHostname              // hostname
	DomainTypeWildcard              // wildcard
)

var validDomainType = map[DomainType]bool{
	DomainTypeCloudflare: true,
	DomainTypeHostname:   true,
	DomainTypeWildcard:   true,
}

func (t DomainType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *DomainType) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	*t = DomainType(0)

	for _, x := range []DomainType{DomainTypeCloudflare, DomainTypeHostname, DomainTypeWildcard} {
		if x.String() == s {
			*t = x
			return nil
		}
	}
	return nil
}

func (t DomainType) Valid() bool {
	return validDomainType[t]
}
