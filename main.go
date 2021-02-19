package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"moonrhythm/deploys/api"
	"moonrhythm/deploys/api/client"
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
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

func help() {
	fmt.Println("deploys.app cli")
}
