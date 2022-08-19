package models

const (
	// TCStatusOK defines the status code OK of the test connection request.
	TCStatusOK = "ok"

	// TCStatusNotice defines the status code "notice" of the test connection request.
	TCStatusNotice = "notice"

	// TCStatusWarning defines the status code "warning" of the test connection request.
	TCStatusWarning = "warning"

	// TCStatusError defines the status code "error" of the test connection request.
	TCStatusError = "error"

	// TCResultOK defines the result without errors of the test connection request.
	TCResultOK = "ok"

	// TCResultConnectionError defines a connection error of the test connection request.
	TCResultConnectionError = "connection_error"

	// TCResultUnexploredImage defines the notice about unexplored Docker image yet.
	TCResultUnexploredImage = "unexplored_image"

	// TCResultMissingExtension defines the warning about a missing extension.
	TCResultMissingExtension = "missing_extension"

	// TCResultMissingLocale defines the warning about a missing locale.
	TCResultMissingLocale = "missing_locale"

	// TCMessageOK defines the source database is ready for dump and restore.
	TCMessageOK = "Database ready for dump and restore"
)

// TestConnection represents the response of the test connection request.
type TestConnection struct {
	Status  string `json:"status"`
	Result  string `json:"result"`
	Message string `json:"message"`
}
