package api

type API interface {
	Me() Me
	Location() Location
	Project() Project
	Role() Role
	Deployment() Deployment
	Disk() Disk
	PullSecret() PullSecret
	Collector() Collector
}
