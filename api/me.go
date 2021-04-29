package api

import (
	"context"
	"mime/multipart"
	"strconv"
)

type Me interface {
	Get(ctx context.Context, _ *Empty) (*MeItem, error)
	Authorized(ctx context.Context, m *MeAuthorized) (*MeAuthorizedResult, error)
	UploadKYCDocument(ctx context.Context, m *MeUploadKYCDocument) (*MeUploadKYCDocumentResult, error)
}

type MeItem struct {
	Email string `json:"email" yaml:"email"`
	KYC   bool   `json:"kyc" yaml:"kyc"`
}

func (m *MeItem) Table() [][]string {
	return [][]string{
		{"EMAIL"},
		{
			m.Email,
		},
	}
}

type MeAuthorized struct {
	ProjectID   int64    `json:"projectId" yaml:"projectId"`
	Project     string   `json:"project" yaml:"project"`
	Permissions []string `json:"permissions" yaml:"permissions"`
}

type MeAuthorizedResult struct {
	Authorized bool `json:"authorized"`
}

func (m *MeAuthorizedResult) Table() [][]string {
	return [][]string{
		{"AUTHORIZED"},
		{
			strconv.FormatBool(m.Authorized),
		},
	}
}

type MeUploadKYCDocument struct {
	File *multipart.FileHeader
}

func (m *MeUploadKYCDocument) UnmarshalMultipartForm(v *multipart.Form) error {
	fps, ok := v.File["document"]
	if !ok {
		return nil
	}
	if len(fps) != 1 {
		return nil
	}
	m.File = fps[0]
	return nil
}

type MeUploadKYCDocumentResult struct {
	DocumentID int64 `json:"documentId" yaml:"documentId"`
}
