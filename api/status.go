package api

type Status int

const (
	Pending Status = iota
	Success
	Error
	Cancelled
	ErrorPendingCleanupResource // TODO: remove ?
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "Pending"
	case Success:
		return "Success"
	case Error:
		return "Error"
	case Cancelled:
		return "Cancelled"
	case ErrorPendingCleanupResource:
		return "Error"
	default:
		return ""
	}
}
