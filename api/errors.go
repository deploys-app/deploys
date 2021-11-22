package api

import (
	"github.com/acoshift/arpc/v2"
)

var (
	ErrUnauthorized                  = newError("api: unauthorized")
	ErrForbidden                     = newError("api: forbidden")
	ErrLocationNotAvailable          = newError("api: location not available")
	ErrLocationNotSupport            = newError("api: location not support")
	ErrSIDNotAvailable               = newError("api: sid not available")
	ErrRoleNotFound                  = newError("api: role not found")
	ErrRoleSIDNotAvailable           = newError("api: role sid not available")
	ErrProjectNotFound               = newError("api: project not found")
	ErrBillingAccountNotFound        = newError("api: billing account not found")
	ErrBillingAccountNotActive       = newError("api: billing account not active, please contact us via email to activate billing account")
	ErrDeploymentNotFound            = newError("api: deployment not found")
	ErrCanNotDelete                  = newError("api: can not delete")
	ErrCanNotPause                   = newError("api: can not pause")
	ErrCanNotResume                  = newError("api: can not resume")
	ErrWorkloadIdentityNotFound      = newError("api: workload identity not found")
	ErrWorkloadIdentityAlreadyExists = newError("api: workload identity already exists")
	ErrWorkloadIdentityInUse         = newError("api: workload identity in use")
	ErrUserNotFound                  = newError("api: user not found")
	ErrDomainNotAvailable            = newError("api: domain not available")
	ErrReplicasInvalid               = newError("api: replicas invalid")
	ErrCanMapOnlyWebService          = newError("api: can not map to deployment other than web service type")
	ErrScheduleInvalid               = newError("api: schedule invalid")
	ErrTypeInvalid                   = newError("api: type invalid")
	ErrTypeNotAllowChange            = newError("api: type not allow to change")
	ErrDiskNotFound                  = newError("api: disk not found")
	ErrDiskSizeMustScaleUp           = newError("api: disk size must scale up")
	ErrDiskAlreadyExists             = newError("api: disk already exists")
	ErrDiskInUsed                    = newError("api: disk in use")
	ErrPullSecretNameNotAvailable    = newError("api: pull secret name not available")
	ErrPullSecretNotFound            = newError("api: pull secret not found")
	ErrPullSecretInUse               = newError("api: pull secret in use")
	ErrServiceAccountNotFound        = newError("api: service account not found")
	ErrServiceAccountAlreadyExists   = newError("api: service account already exists")
	ErrMaximumDeploymentReach        = newError("api: maximum deployment reach")
	ErrRouteNotFound                 = newError("api: route not found")
)

var AllErrors []error

func newError(msg string) error {
	err := arpc.NewError(msg)
	AllErrors = append(AllErrors, err)
	return err
}
