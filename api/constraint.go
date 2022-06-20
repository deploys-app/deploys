package api

import (
	"regexp"
)

const (
	ReValidNameStr     = `^[a-z][a-z0-9\-]*[a-z0-9]$`
	ReValidScheduleStr = `^((((\*(/\d+)?)|(\d+((-\d+)|(/\d+))?)),?)+\s?){5}$`
)

// global
var (
	ReValidName     = regexp.MustCompile(ReValidNameStr)
	ReValidSchedule = regexp.MustCompile(ReValidScheduleStr)
)

// global
const (
	MinNameLength = 3
	MaxNameLength = 27
)

// Deployments
const (
	DeploymentMinReplicas = 1
	DeploymentMaxReplicas = 20
	DiskMaxSize           = 100
)
