package types

import "fmt"

type MyError struct {
	Message string
}

func (e *MyError) Error() string {
	return fmt.Sprintf("Vesting Error: %s", e.Message)
}
