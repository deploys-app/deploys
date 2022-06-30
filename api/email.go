package api

import (
	"context"
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
	Project string    `json:"project" yaml:"project"`
	From    string    `json:"from" yaml:"from"`
	To      []string  `json:"to" yaml:"to"`
	Subject string    `json:"subject" yaml:"subject"`
	Body    EmailBody `json:"body" yaml:"body"`
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
	if v.Must(m.From != "", "from required") {
		v.Must(govalidator.IsEmail(m.From), "from invalid")
	}
	if v.Must(len(m.To) > 0, "to required") {
		for _, to := range m.To {
			v.Mustf(govalidator.IsEmail(to), "to '%s' invalid", to)
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
