package api

type API interface {
	Me() Me
	Location() Location
	Project() Project
	Role() Role
	Deployment() Deployment
	Route() Route
	Disk() Disk
	PullSecret() PullSecret
	Collector() Collector
}
