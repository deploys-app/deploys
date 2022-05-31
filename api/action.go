package api

import (
	"encoding/json"
)

//go:generate stringer -type=Action -linecomment
type Action int

const (
	_      Action = iota
	Create        // create
	Delete        // delete
)

func (a Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Action) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	*a = Action(0)

	for _, x := range []Action{Create, Delete} {
		if x.String() == s {
			*a = x
			return nil
		}
	}
	return nil
}
