package api

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/moonrhythm/validator"
)

type Deployment interface {
	Deploy(ctx context.Context, m *DeploymentDeploy) (*Empty, error)
	List(ctx context.Context, m *DeploymentList) (*DeploymentListResult, error)
	Get(ctx context.Context, m *DeploymentGet) (*DeploymentGetResult, error)
	Revisions(ctx context.Context, m *DeploymentRevisions) (*DeploymentRevisionsResult, error)
	Resume(ctx context.Context, m *DeploymentResume) (*Empty, error)
	Pause(ctx context.Context, m *DeploymentPause) (*Empty, error)
	Rollback(ctx context.Context, m *DeploymentRollback) (*Empty, error)
	Delete(ctx context.Context, m *DeploymentDelete) (*Empty, error)
	Metrics(ctx context.Context, m *DeploymentMetrics) (*DeploymentMetricsResult, error)
}

type DeploymentType int

const (
	_ DeploymentType = iota
	DeploymentTypeWebService
	DeploymentTypeWorker
	DeploymentTypeCronJob
	DeploymentTypeTCPService
	DeploymentTypeInternalTCPService
)

var allDeploymentTypes = []DeploymentType{
	DeploymentTypeWebService,
	DeploymentTypeWorker,
	DeploymentTypeCronJob,
	DeploymentTypeTCPService,
	DeploymentTypeInternalTCPService,
}

var validDeploymentType = func() map[DeploymentType]bool {
	m := map[DeploymentType]bool{}
	for _, t := range allDeploymentTypes {
		m[t] = true
	}
	return m
}()

func ParseDeploymentTypeString(s string) DeploymentType {
	for _, t := range allDeploymentTypes {
		if t.String() == s {
			return t
		}
	}
	return 0
}

func (t DeploymentType) String() string {
	switch t {
	case DeploymentTypeWebService:
		return "WebService"
	case DeploymentTypeWorker:
		return "Worker"
	case DeploymentTypeCronJob:
		return "CronJob"
	case DeploymentTypeTCPService:
		return "TCPService"
	case DeploymentTypeInternalTCPService:
		return "InternalTCPService"
	default:
		return ""
	}
}

func (t DeploymentType) Text() string {
	switch t {
	case DeploymentTypeWebService:
		return "Web Service"
	case DeploymentTypeWorker:
		return "Worker"
	case DeploymentTypeCronJob:
		return "CronJob"
	case DeploymentTypeTCPService:
		return "TCP Service"
	case DeploymentTypeInternalTCPService:
		return "Internal TCP Service"
	default:
		return ""
	}
}

func (t DeploymentType) Int() int {
	return int(t)
}

func (t DeploymentType) IsZero() bool {
	return t == 0
}

func (t DeploymentType) Valid() bool {
	// zero value is valid
	if t == 0 {
		return true
	}
	return validDeploymentType[t]
}

func (t *DeploymentType) parseString(s string) error {
	if s == "" {
		*t = 0
		return nil
	}
	*t = ParseDeploymentTypeString(s)
	if t.IsZero() {
		return fmt.Errorf("invalid deployment type")
	}
	return nil
}

func (t DeploymentType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *DeploymentType) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	return t.parseString(s)
}

func (t DeploymentType) MarshalYAML() (interface{}, error) {
	return t.String(), nil
}

func (t *DeploymentType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	return t.parseString(s)
}

func (t DeploymentType) HasExternalTCPAddress() bool {
	switch t {
	default:
		return false
	case DeploymentTypeTCPService:
		return true
	}
}

func (t DeploymentType) HasInternalTCPAddress() bool {
	switch t {
	default:
		return false
	case DeploymentTypeWebService:
		return true
	case DeploymentTypeTCPService:
		return true
	case DeploymentTypeInternalTCPService:
		return true
	}
}

type ResourceItem struct {
	// CPU    string `json:"cpu" yaml:"cpu"`
	Memory string `json:"memory" yaml:"memory"`
}

type DeploymentResource struct {
	Requests ResourceItem `json:"requests" yaml:"requests"`
	Limits   ResourceItem `json:"limits" yaml:"limits"`
}

type DeploymentDeploy struct {
	Project          string              `json:"project"`
	Location         string              `json:"location"`
	Name             string              `json:"name"`
	Image            string              `json:"image"`
	MinReplicas      *int                `json:"minReplicas"`
	MaxReplicas      *int                `json:"maxReplicas"`
	Type             DeploymentType      `json:"type"`
	Port             *int                `json:"port"`
	Env              map[string]string   `json:"env"`       // override all env
	AddEnv           map[string]string   `json:"addEnv"`    // add env to old revision env
	RemoveEnv        []string            `json:"removeEnv"` // remove env from old revision env
	Command          []string            `json:"command"`
	Args             []string            `json:"args"`
	WorkloadIdentity *string             `json:"workloadIdentity"` // workload identity name
	PullSecret       *string             `json:"pullSecret"`       // pull secret name
	Disk             *DeploymentDisk     `json:"disk"`             // type=Stateful
	Schedule         *string             `json:"schedule"`         // type=CronJob
	Resources        *DeploymentResource `json:"resources"`
}

type DeploymentDisk struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	SubPath   string `json:"subPath"`
}

func (m *DeploymentDeploy) Valid() error {
	m.Name = strings.TrimSpace(m.Name)
	m.Image = strings.ReplaceAll(m.Image, " ", "") // remove all space in image
	// m.Image = strings.ToLower(m.Image) // image tag can be lowercase

	v := validator.New()

	v.Must(m.Project != "", "project required")
	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	{
		cnt := utf8.RuneCountInString(m.Name)
		v.Mustf(cnt >= MinNameLength && cnt <= MaxNameLength, "name must have length between %d-%d characters", MinNameLength, MaxNameLength)
	}
	v.Must(m.Image != "", "image required")

	// validate replicas if provided
	if m.MinReplicas != nil {
		v.Mustf(*m.MinReplicas >= 0 && *m.MinReplicas <= DeploymentMaxReplicas, "min replicas value must be in range [%d, %d]", 0, DeploymentMaxReplicas)
	}
	if m.MaxReplicas != nil {
		v.Mustf(*m.MaxReplicas >= 0 && *m.MaxReplicas <= DeploymentMaxReplicas, "max replicas value must be in range [%d, %d]", 0, DeploymentMaxReplicas)
	}
	if m.MinReplicas != nil && m.MaxReplicas != nil {
		v.Must(*m.MinReplicas <= *m.MaxReplicas, "max replicas must higher or equal min replicas")
	}

	// feature not support autoscaling
	if m.MinReplicas != nil && m.MaxReplicas != nil && *m.MinReplicas != *m.MaxReplicas {
		v.Mustf(m.Disk == nil, "using disk not support auto-scaling")
	}

	// validate disk
	if m.Disk != nil {
		v.Mustf(m.Disk.Name != "", "disk name required")
		v.Mustf(m.Disk.MountPath != "", "disk mount path required")
		if m.Disk.SubPath != "" {
			v.Mustf(!filepath.IsAbs(m.Disk.SubPath), "disk sub path must be absolute path")
		}
	}

	// validate type
	if !m.Type.IsZero() {
		v.Must(m.Type.Valid(), "invalid type")

		switch m.Type {
		case DeploymentTypeWebService:
			if v.Must(m.Port != nil, "port required") {
				v.Must(*m.Port > 0, "invalid port")
			}
		case DeploymentTypeCronJob:
			if m.Schedule != nil {
				if v.Must(*m.Schedule != "", "schedule required") {
					v.Must(ReValidSchedule.MatchString(*m.Schedule), "schedule invalid")
				}
			}
		case DeploymentTypeTCPService:
			if v.Must(m.Port != nil, "port required") {
				v.Must(*m.Port > 0, "invalid port")
			}
		case DeploymentTypeInternalTCPService:
			if v.Must(m.Port != nil, "port required") {
				v.Must(*m.Port > 0, "invalid port")
			}
		}
	}

	return WrapValidate(v)
}

type DeploymentList struct {
	Project string `json:"project"`
}

func (m *DeploymentList) Valid() error {
	v := validator.New()

	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type DeploymentListResult struct {
	Deployments []*DeploymentItem `json:"deployments" yaml:"deployments"`
}

func (m *DeploymentListResult) Table() [][]string {
	table := [][]string{
		{"NAME", "TYPE", "STATUS", "AGE"},
	}
	for _, x := range m.Deployments {
		table = append(table, []string{
			x.Name,
			x.Type.String(),
			x.Status.String(),
			age(x.CreatedAt),
		})
	}
	return table
}

type DeploymentItem struct {
	Project     string             `json:"project" yaml:"project"`
	Location    string             `json:"location" yaml:"location"`
	Name        string             `json:"name" yaml:"name"`
	Type        DeploymentType     `json:"type" yaml:"type"`
	Image       string             `json:"image" yaml:"image"`
	Revision    int64              `json:"revision" yaml:"revision"`
	Resources   DeploymentResource `json:"resources" yaml:"resources"`
	MinReplicas int                `json:"minReplicas" yaml:"minReplicas"`
	MaxReplicas int                `json:"maxReplicas" yaml:"maxReplicas"`
	Status      Status             `json:"status" yaml:"status"`
	Action      DeploymentAction   `json:"action" yaml:"action"`
	CreatedAt   time.Time          `json:"createdAt" yaml:"createdAt"`
	CreatedBy   string             `json:"createdBy" yaml:"createdBy"`
	SuccessAt   time.Time          `json:"successAt" yaml:"successAt"`
}

type DeploymentGet struct {
	Project  string `json:"project"`
	Name     string `json:"name"`
	Revision int    `json:"revision"` // 0 = latest
}

func (m *DeploymentGet) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	v.Mustf(utf8.RuneCountInString(m.Name) <= MaxNameLength, "name must have length less then %d characters", MaxNameLength)
	v.Must(m.Project != "", "project required")
	v.Must(m.Revision >= 0, "invalid revision")

	return WrapValidate(v)
}

type DeploymentGetResult struct {
	Project          string             `json:"project" yaml:"project"`
	Location         string             `json:"location" yaml:"location"`
	Name             string             `json:"name" yaml:"name"`
	Type             DeploymentType     `json:"type" yaml:"type"`
	Revision         int64              `json:"revision" yaml:"revision"`
	Image            string             `json:"image" yaml:"image"`
	Env              map[string]string  `json:"env" yaml:"env"`
	Command          []string           `json:"command" yaml:"command"`
	Args             []string           `json:"args" yaml:"args"`
	WorkloadIdentity string             `json:"workloadIdentity" yaml:"workloadIdentity"`
	PullSecret       string             `json:"pullSecret" yaml:"pullSecret"`
	Disk             *DeploymentDisk    `json:"disk" yaml:"disk"`
	MinReplicas      int                `json:"minReplicas" yaml:"minReplicas"`
	MaxReplicas      int                `json:"maxReplicas" yaml:"maxReplicas"`
	Schedule         string             `json:"schedule" yaml:"schedule"`
	Port             int                `json:"port" yaml:"port"`
	NodePort         int                `json:"nodePort" yaml:"nodePort"`
	Annotations      map[string]string  `json:"annotations" yaml:"annotations"`
	Resources        DeploymentResource `json:"resources" yaml:"resources"`
	URL              string             `json:"url" yaml:"url"`
	LogURL           string             `json:"logUrl" yaml:"logUrl"`
	EventURL         string             `json:"eventUrl" yaml:"eventUrl"`
	Address          string             `json:"address" yaml:"address"`
	InternalAddress  string             `json:"internalAddress" yaml:"internalAddress"`
	Status           Status             `json:"status" yaml:"status"`
	Action           DeploymentAction   `json:"action" yaml:"action"`
	AllocatedPrice   float64            `json:"allocatedPrice" yaml:"allocatedPrice"`
	CreatedAt        time.Time          `json:"createdAt" yaml:"createdAt"`
	CreatedBy        string             `json:"createdBy" yaml:"createdBy"`
	SuccessAt        time.Time          `json:"successAt" yaml:"successAt"`
}

type DeploymentRevisions struct {
	Project string `json:"project"`
	Name    string `json:"name"`
}

func (m *DeploymentRevisions) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	v.Mustf(utf8.RuneCountInString(m.Name) <= MaxNameLength, "name must have length less then %d characters", MaxNameLength)
	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type DeploymentRevisionsResult struct {
	Items []*DeploymentItem `json:"items" yaml:"items"`
}

type DeploymentResume struct {
	Project string `json:"project"`
	Name    string `json:"name"`
}

func (m *DeploymentResume) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	v.Mustf(utf8.RuneCountInString(m.Name) <= MaxNameLength, "name must have length less then %d characters", MaxNameLength)
	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type DeploymentPause struct {
	Project string `json:"project"`
	Name    string `json:"name"`
}

func (m *DeploymentPause) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	v.Mustf(utf8.RuneCountInString(m.Name) <= MaxNameLength, "name must have length less then %d characters", MaxNameLength)
	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type DeploymentRollback struct {
	Project  string `json:"project"`
	Name     string `json:"name"`
	Revision int    `json:"revision"`
}

func (m *DeploymentRollback) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	v.Mustf(utf8.RuneCountInString(m.Name) <= MaxNameLength, "name must have length less then %d characters", MaxNameLength)
	v.Must(m.Project != "", "project required")
	v.Must(m.Revision >= 1, "invalid revision")

	return WrapValidate(v)
}

type DeploymentDelete struct {
	Project string `json:"project"`
	Name    string `json:"name"`
}

func (m *DeploymentDelete) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	v.Mustf(utf8.RuneCountInString(m.Name) <= MaxNameLength, "name must have length less then %d characters", MaxNameLength)
	v.Must(m.Project != "", "project required")

	return WrapValidate(v)
}

type DeploymentMetrics struct {
	Project   string                     `json:"project" yaml:"project"`
	Name      string                     `json:"name" yaml:"name"`
	TimeRange DeploymentMetricsTimeRange `json:"timeRange" yaml:"timeRange"`
}

type DeploymentMetricsTimeRange string

const (
	DeploymentMetricsTimeRange1h     = "1h"
	DeploymentMetricsTimeRange6h     = "6h"
	DeploymentMetricsTimeRange12h    = "12h"
	DeploymentMetricsTimeRange1d     = "1d"
	DeploymentMetricsTimeRange1hagg  = "1hagg"
	DeploymentMetricsTimeRange6hagg  = "6hagg"
	DeploymentMetricsTimeRange12hagg = "12hagg"
	DeploymentMetricsTimeRange1dagg  = "1dagg"
	DeploymentMetricsTimeRange2dagg  = "2dagg"
	DeploymentMetricsTimeRange7dagg  = "7dagg"
	DeploymentMetricsTimeRange30dagg = "30dagg"
)

var allDeploymentMetricsTimeRange = []DeploymentMetricsTimeRange{
	DeploymentMetricsTimeRange1h,
	DeploymentMetricsTimeRange6h,
	DeploymentMetricsTimeRange12h,
	DeploymentMetricsTimeRange1d,
	DeploymentMetricsTimeRange1hagg,
	DeploymentMetricsTimeRange6hagg,
	DeploymentMetricsTimeRange12hagg,
	DeploymentMetricsTimeRange1dagg,
	DeploymentMetricsTimeRange2dagg,
	DeploymentMetricsTimeRange7dagg,
	DeploymentMetricsTimeRange30dagg,
}

var validDeploymentMetricsTimeRange = func() map[DeploymentMetricsTimeRange]bool {
	m := map[DeploymentMetricsTimeRange]bool{}
	for _, t := range allDeploymentMetricsTimeRange {
		m[t] = true
	}
	return m
}()

func (m *DeploymentMetrics) Valid() error {
	m.Name = strings.TrimSpace(m.Name)

	v := validator.New()

	v.Must(ReValidName.MatchString(m.Name), "name invalid "+ReValidNameStr)
	v.Mustf(utf8.RuneCountInString(m.Name) <= MaxNameLength, "name must have length less then %d characters", MaxNameLength)
	v.Must(m.Project != "", "project required")
	v.Must(validDeploymentMetricsTimeRange[m.TimeRange], "timeRange invalid")

	return WrapValidate(v)
}

type DeploymentMetricsResult struct {
	CPUUsage    []*DeploymentMetricsLine `json:"cpuUsage"`
	MemoryUsage []*DeploymentMetricsLine `json:"memoryUsage"`
	Memory      []*DeploymentMetricsLine `json:"memory"`
	Requests    []*DeploymentMetricsLine `json:"requests"`
	Egress      []*DeploymentMetricsLine `json:"egress"`
}

type DeploymentMetricsLine struct {
	Name   string       `json:"name" yaml:"name"`
	Points [][2]float64 `json:"points" yaml:"points"`
}
