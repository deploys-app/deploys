package api

import (
	"regexp"
)

const (
	ReValidNameStr     = `^[a-z][a-z0-9\-]*[^\-]$`
	ReValidScheduleStr = `^((((\*(/\d+)?)|(\d+((-\d+)|(/\d+))?)),?)+\s?){5}$`
)

// global
var (
	ReValidName     = regexp.MustCompile(ReValidNameStr)
	ReValidSchedule = regexp.MustCompile(ReValidScheduleStr)
)

// global
const (
	MinNameLength = 4
	MaxNameLength = 27
)

// Deployments
const (
	DeploymentMinReplicas = 1
	DeploymentMaxReplicas = 6
)
