package api

import "encoding/json"

//go:generate stringer -type=DomainStatus -linecomment
type DomainStatus int

const (
	DomainStatusPending DomainStatus = iota // pending
	DomainStatusSuccess                     // success
	DomainStatusError                       // error
	DomainStatusVerify                      // verify
)

var allDomainStatus = []DomainStatus{
	DomainStatusPending,
	DomainStatusSuccess,
	DomainStatusError,
	DomainStatusVerify,
}

func parseDomainStatus(s string) DomainStatus {
	for _, x := range allDomainStatus {
		if x.String() == s {
			return x
		}
	}
	return DomainStatusPending
}

func (s DomainStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *DomainStatus) UnmarshalJSON(b []byte) error {
	var t string
	err := json.Unmarshal(b, &t)
	if err != nil {
		return err
	}

	*s = parseDomainStatus(t)
	return nil
}

func (s DomainStatus) MarshalYAML() (any, error) {
	return s.String(), nil
}

func (s *DomainStatus) UnmarshalYAML(unmarshal func(any) error) error {
	var t string
	err := unmarshal(&t)
	if err != nil {
		return err
	}
	*s = parseDomainStatus(t)
	return nil
}
