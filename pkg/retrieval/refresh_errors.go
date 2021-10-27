/*
2021 © Postgres.ai
*/

package retrieval

// SkipRefreshingError defines an error when data refreshing is skipped.
type SkipRefreshingError struct {
	msg string
}

// NewSkipRefreshingError creates a new SkipRefreshingError.
func NewSkipRefreshingError(msg string) *SkipRefreshingError {
	return &SkipRefreshingError{
		msg: msg,
	}
}

// Error returns error message.
func (e *SkipRefreshingError) Error() string {
	return e.msg
}
