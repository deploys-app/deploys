package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/moonrhythm/validator"
)

type Email interface {
	Send(ctx context.Context, m *EmailSend) (*Empty, error)
	List(ctx context.Context, m *EmailList) (*EmailListResult, error)
}

type EmailSend struct {
	Project string      `json:"project" yaml:"project"`
	From    EmailAddr   `json:"from" yaml:"from"`
	To      []EmailAddr `json:"to" yaml:"to"`
	Subject string      `json:"subject" yaml:"subject"`
	Body    EmailBody   `json:"body" yaml:"body"`
}

type EmailAddr struct {
	Email string `json:"email" yaml:"email"`
	Name  string `json:"name" yaml:"name"`
}

func (a *EmailAddr) Valid() bool {
	a.Email = strings.TrimSpace(a.Email)
	a.Name = strings.TrimSpace(a.Name)

	if a.Email == "" {
		return false
	}
	if !govalidator.IsEmail(a.Email) {
		return false
	}
	return true
}

func (a *EmailAddr) IsZero() bool {
	return a.Email == ""
}

func (a EmailAddr) Address() string {
	if a.Name == "" {
		return a.Email
	}
	return fmt.Sprintf("%s <%s>", a.Name, a.Email)
}

func (a *EmailAddr) UnmarshalJSON(b []byte) error {
	*a = EmailAddr{}

	if len(b) == 0 {
		return fmt.Errorf("empty address")
	}

	switch string(b[0]) {
	case "{":
		var t struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		err := json.Unmarshal(b, &t)
		if err != nil {
			return err
		}
		a.Email = t.Email
		a.Name = t.Name
	case "\"":
		var t string
		err := json.Unmarshal(b, &t)
		if err != nil {
			return err
		}
		a.Email = t
	default:
		return fmt.Errorf("invalid address")
	}
	return nil
}

type EmailBody struct {
	Type    EmailType `json:"type" yaml:"type"`
	Content string    `json:"content" yaml:"content"`
}

type EmailType string

const (
	EmailTypeText EmailType = "text/plain"
	EmailTypeHTML EmailType = "text/html"
)

func (t EmailType) Valid() bool {
	return t == EmailTypeText ||
		t == EmailTypeHTML
}

func (m *EmailSend) Valid() error {
	v := validator.New()

	m.Subject = strings.TrimSpace(m.Subject)

	v.Must(m.Project != "", "project required")
	if v.Must(!m.From.IsZero(), "from required") {
		v.Must(m.From.Valid(), "from invalid")
	}
	if v.Must(len(m.To) > 0, "to required") {
		for _, to := range m.To {
			v.Mustf(to.Valid(), "to '%s' invalid", to.Address())
		}
	}
	v.Must(m.Subject != "", "subject required")
	v.Must(m.Body.Type.Valid(), "body.type invalid")
	v.Must(m.Body.Content != "", "body.content required")

	return WrapValidate(v)
}

type EmailItem struct {
	Domain    string    `json:"domain" yaml:"domain"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
}

type EmailList struct {
	Project string `json:"project" yaml:"project"`
}

func (m *EmailList) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type EmailListResult struct {
	Items []*EmailItem `json:"items"`
}
