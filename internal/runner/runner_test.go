package runner

import (
	"io"
	"os"
	"testing"

	"github.com/deploys-app/api"
)

// print's toon case must render the same encoding/json-tagged data as -ojson,
// just in the TOON encoding (github.com/moonrhythm/toon), plus a trailing
// newline like the other output modes.
func TestPrint_Toon(t *testing.T) {
	type item struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	type wrapper struct {
		Total int    `json:"total"`
		Items []item `json:"items"`
	}

	tmp, err := os.CreateTemp(t.TempDir(), "toon")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()

	rn := Runner{Output: tmp, OutputMode: "toon"}
	v := wrapper{
		Total: 2,
		Items: []item{
			{Name: "alice", Age: 30},
			{Name: "bob", Age: 25},
		},
	}
	if err := rn.print(v); err != nil {
		t.Fatalf("print: %v", err)
	}

	b, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	want := "total: 2\nitems[2]{name,age}:\n  alice,30\n  bob,25\n"
	if string(b) != want {
		t.Errorf("print(toon) = %q; want %q", string(b), want)
	}
}

// -otoon is a shorthand for --output=toon, rewritten before flag parsing (like
// the other -o* shorthands).
func TestReplaceShortFlag_Toon(t *testing.T) {
	var rn Runner
	args := []string{"-otoon", "list"}
	rn.replaceShortFlag(args)
	if args[0] != "--output=toon" {
		t.Errorf("replaceShortFlag(-otoon) = %q; want --output=toon", args[0])
	}
}

// Omitted optional flags must leave their request fields nil/empty so a deploy
// is a merge that preserves the previous revision's values.
func TestParseDeploymentDeploy_OmittedStayNil(t *testing.T) {
	req, out, err := parseDeploymentDeploy(io.Discard, []string{
		"-project", "p", "-location", "l", "-name", "web", "-image", "img:1",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != "table" {
		t.Errorf("output = %q; want table", out)
	}
	if req.Project != "p" || req.Location != "l" || req.Name != "web" || req.Image != "img:1" {
		t.Errorf("basics not mapped: %+v", req)
	}
	if req.Port != nil || req.MinReplicas != nil || req.MaxReplicas != nil ||
		req.Protocol != nil || req.Internal != nil || req.WorkloadIdentity != nil ||
		req.PullSecret != nil || req.Schedule != nil || req.TTL != nil ||
		req.Disk != nil || req.Resources != nil || req.Access != nil {
		t.Errorf("omitted pointers should be nil: %+v", req)
	}
	if req.Env != nil || req.AddEnv != nil || req.MountData != nil ||
		req.RemoveEnv != nil || req.EnvGroups != nil || req.Command != nil || req.Args != nil {
		t.Errorf("omitted maps/slices should be nil: %+v", req)
	}
	if req.Type != 0 {
		t.Errorf("omitted type should be the zero (unset) value, got %v", req.Type)
	}
}

// Backward compatibility: the long-standing numeric flags keep their >0 guard
// (so -port 0 stays nil, matching the previous inline implementation).
func TestParseDeploymentDeploy_NumericBackcompat(t *testing.T) {
	req, _, err := parseDeploymentDeploy(io.Discard, []string{"-port", "8080", "-minReplicas", "1", "-maxReplicas", "5"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if req.Port == nil || *req.Port != 8080 || req.MinReplicas == nil || *req.MinReplicas != 1 || req.MaxReplicas == nil || *req.MaxReplicas != 5 {
		t.Errorf("numeric flags not mapped: %+v", req)
	}

	req, _, err = parseDeploymentDeploy(io.Discard, []string{"-port", "0"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if req.Port != nil {
		t.Errorf("-port 0 should stay nil (>0 guard), got %v", *req.Port)
	}
}

// visitedFlags semantics: an explicitly-passed zero/false must be sent (so a
// user can clear a TTL or set internal=false), while omitting keeps it nil.
func TestParseDeploymentDeploy_VisitClears(t *testing.T) {
	req, _, err := parseDeploymentDeploy(io.Discard, []string{"-ttl", "0", "-internal=false"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if req.TTL == nil || *req.TTL != 0 {
		t.Errorf("-ttl 0 should send a zero pointer (clear), got %v", req.TTL)
	}
	if req.Internal == nil || *req.Internal != false {
		t.Errorf("-internal=false should send a false pointer, got %v", req.Internal)
	}

	req, _, _ = parseDeploymentDeploy(io.Discard, []string{"-ttl", "3600", "-internal"})
	if req.TTL == nil || *req.TTL != 3600 || req.Internal == nil || *req.Internal != true {
		t.Errorf("set ttl/internal not mapped: %+v", req)
	}
}

func TestParseDeploymentDeploy_MapsListsAndStructs(t *testing.T) {
	req, _, err := parseDeploymentDeploy(io.Discard, []string{
		"-type", "Worker",
		"-env", "A=1", "-env", "B=2",
		"-addEnv", "C=3",
		"-removeEnv", "D,E",
		"-envGroups", "g1,g2",
		"-command", "/bin/app",
		"-args", "--a,--b=1",
		"-workloadIdentity", "wi",
		"-pullSecret", "ps",
		"-protocol", "h2c",
		"-diskName", "data", "-diskMountPath", "/data",
		"-cpuRequest", "250m", "-memLimit", "512Mi",
		"-requireGoogleLogin", "-allowedDomains", "example.com",
		"-mountData", "/etc/cfg=hello",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if req.Type != api.DeploymentTypeWorker {
		t.Errorf("type = %v; want Worker", req.Type)
	}
	if req.Env["A"] != "1" || req.Env["B"] != "2" || req.AddEnv["C"] != "3" {
		t.Errorf("env maps = %v / %v", req.Env, req.AddEnv)
	}
	if len(req.RemoveEnv) != 2 || req.RemoveEnv[0] != "D" || len(req.EnvGroups) != 2 {
		t.Errorf("lists = %v / %v", req.RemoveEnv, req.EnvGroups)
	}
	if len(req.Command) != 1 || req.Command[0] != "/bin/app" || len(req.Args) != 2 || req.Args[1] != "--b=1" {
		t.Errorf("command/args = %v / %v", req.Command, req.Args)
	}
	if req.WorkloadIdentity == nil || *req.WorkloadIdentity != "wi" || req.PullSecret == nil || *req.PullSecret != "ps" {
		t.Errorf("wi/pullSecret not mapped: %+v", req)
	}
	if req.Protocol == nil || *req.Protocol != api.DeploymentProtocolH2C {
		t.Errorf("protocol not mapped: %v", req.Protocol)
	}
	if req.Disk == nil || req.Disk.Name != "data" || req.Disk.MountPath != "/data" {
		t.Errorf("disk not mapped: %+v", req.Disk)
	}
	if req.Resources == nil || req.Resources.Requests.CPU != "250m" || req.Resources.Limits.Memory != "512Mi" {
		t.Errorf("resources not mapped: %+v", req.Resources)
	}
	if req.Access == nil || !req.Access.RequireGoogleLogin || len(req.Access.AllowedDomains) != 1 {
		t.Errorf("access not mapped: %+v", req.Access)
	}
	if req.MountData["/etc/cfg"] != "hello" {
		t.Errorf("mountData not mapped: %v", req.MountData)
	}
}

func TestParseDeploymentDeploy_Errors(t *testing.T) {
	if _, _, err := parseDeploymentDeploy(io.Discard, []string{"-env", "BAD"}); err == nil {
		t.Error("invalid -env KEY=VALUE should error")
	}
	if _, _, err := parseDeploymentDeploy(io.Discard, []string{"-nope"}); err == nil {
		t.Error("unknown flag should return an error, not exit")
	}
	if _, out, err := parseDeploymentDeploy(io.Discard, []string{"-output", "json"}); err != nil || out != "json" {
		t.Errorf("output passthrough: out=%q err=%v", out, err)
	}
}
