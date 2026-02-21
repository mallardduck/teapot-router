package testutil

import (
	"fmt"
)

// CapturePanic executes the function f and returns the panic message if one occurred.
func CapturePanic(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	f()
	return msg
}
