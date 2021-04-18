package api

type Interface interface {
	Me() Me
	Billing() Billing
	Location() Location
	Project() Project
	Role() Role
	Deployment() Deployment
	Route() Route
	Disk() Disk
	PullSecret() PullSecret
	WorkloadIdentity() WorkloadIdentity
	ServiceAccount() ServiceAccount
	Collector() Collector
}
