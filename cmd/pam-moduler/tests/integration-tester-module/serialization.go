package main

import (
	"encoding/gob"

	"github.com/msteinert/pam"
	"github.com/msteinert/pam/cmd/pam-moduler/tests/internal/utils"
)

// SerializableTransactionError represents a [pam.TransactionError] in a
// serializable way.
type SerializableTransactionError struct {
	Msg       string
	RetStatus pam.Error
}

// Status exposes the [pam.Error] for the serializable error.
func (e *SerializableTransactionError) Status() pam.Error {
	return e.RetStatus
}

func (e *SerializableTransactionError) Error() string {
	return e.Status().Error()
}

func init() {
	gob.Register(map[string]string{})
	gob.Register(Request{})
	gob.Register(pam.Item(0))
	gob.Register(pam.Error(0))
	gob.RegisterName("main.SerializableTransactionError",
		SerializableTransactionError{})
	gob.Register(utils.SerializableError{})
}
