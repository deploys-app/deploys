package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2/google"

	"github.com/deploys-app/deploys/api"
	"github.com/deploys-app/deploys/api/client"
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
	)

	apiClient := &client.Client{
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

	rn := api.Runner{
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
	fmt.Println("deploys.app cli")
}
