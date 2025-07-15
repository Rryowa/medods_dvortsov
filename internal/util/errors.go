package util

import "fmt"

type MyResponseError struct {
	Msg    string
	Status int
}

func (e MyResponseError) Error() string { return e.Msg }

func NewResponseError(status int, format string, args ...interface{}) error {
	return MyResponseError{
		Msg:    fmt.Sprintf(format, args...),
		Status: status,
	}
}
