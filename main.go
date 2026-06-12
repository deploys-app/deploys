package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/deploys-app/api/client"
	"golang.org/x/oauth2/google"

	"github.com/deploys-app/deploys/internal/runner"
)

func main() {
	args := os.Args
	if len(args) <= 1 {
		help()
		return
	}

	var (
		token    = os.Getenv("DEPLOYS_TOKEN")
		authUser = os.Getenv("DEPLOYS_AUTH_USER")
		authPass = os.Getenv("DEPLOYS_AUTH_PASS")
		endpoint = os.Getenv("DEPLOYS_ENDPOINT")
	)

	apiClient := &client.Client{
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	if apiClient.Auth == nil && authUser != "" && authPass != "" {
		apiClient.Auth = func(r *http.Request) {
			r.SetBasicAuth(authUser, authPass)
		}
	}

	if apiClient.Auth == nil && token == "" {
		token, _ = getDefaultToken()
	}

	if apiClient.Auth == nil && token != "" {
		apiClient.Auth = func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+token)
		}
	}

	rn := runner.Runner{
		API:    apiClient,
		Output: os.Stdout,
	}

	err := rn.Run(args[1:]...)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getDefaultToken() (string, error) {
	cred, err := google.FindDefaultCredentials(context.Background())
	if err != nil {
		return "", err
	}

	tk, err := cred.TokenSource.Token()
	if err != nil {
		return "", err
	}

	return tk.AccessToken, nil
}

func help() {
	fmt.Print(`deploys.app cli

Usage:
  deploys <command> <subcommand> [flags]

Commands:
  me                      get, authorized
  billing                 create, list, get, update, delete, report, skus, project,
                          invoices, invoice, downloadinvoice, downloadreceipt
  location                list, get
  project                 create, list, get, update, delete, usage
  role                    create, list, get, delete, grant, revoke, users, bind
  deployment, deploy, d   list, get, deploy, delete, revisions, pause, resume,
                          rollback, metrics, set
  domain                  create, get, list, delete, purgecache
  route                   create, get, list, delete
  waf                     get, list, set, delete, metrics, limitmetrics
  disk                    create, get, list, update, delete
  pullsecret, ps          create, get, list, delete
  workloadidentity, wi    create, get, list, delete
  serviceaccount, sa      create, get, list, update, delete, createkey, deletekey
  email                   send, list
  registry                list, get, tags, manifests, storage, delete,
                          deletemanifest, untag, metrics
  envgroup, eg            create, get, list, update, delete
  auditlog                list
  dropbox                 list, metrics
  github                  link, unlink, list

Flags:
  -output table|yaml|json (or -oyaml, -ojson, -otable)

Environment:
  DEPLOYS_TOKEN           api token
  DEPLOYS_ENDPOINT        override api endpoint
`)
}
