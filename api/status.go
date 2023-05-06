package api

import (
	"encoding/json"
)

type Status int

const (
	Pending Status = iota
	Success
	Error
	Cancelled
	ErrorPendingCleanupResource // TODO: remove ?
	StatusNone
)

var allStatus = []Status{
	Pending,
	Success,
	Error,
	Cancelled,
	ErrorPendingCleanupResource,
	StatusNone,
}

var statusString = map[Status]string{
	Pending:                     "pending",
	Success:                     "success",
	Error:                       "error",
	Cancelled:                   "cancelled",
	ErrorPendingCleanupResource: "error",
}

var statusText = map[Status]string{
	Pending:                     "Pending",
	Success:                     "Success",
	Error:                       "Error",
	Cancelled:                   "Cancelled",
	ErrorPendingCleanupResource: "Error",
}

func parseStatus(s string) Status {
	for _, x := range allStatus {
		if x.String() == s {
			return x
		}
	}
	return StatusNone
}

func (s Status) String() string {
	return statusString[s]
}

func (s Status) Text() string {
	return statusText[s]
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Status) UnmarshalJSON(b []byte) error {
	var t string
	err := json.Unmarshal(b, &t)
	if err != nil {
		return err
	}

	*s = parseStatus(t)
	return nil
}

func (s Status) MarshalYAML() (any, error) {
	return s.String(), nil
}

func (s *Status) UnmarshalYAML(unmarshal func(any) error) error {
	var t string
	err := unmarshal(&t)
	if err != nil {
		return err
	}
	*s = parseStatus(t)
	return nil
}
