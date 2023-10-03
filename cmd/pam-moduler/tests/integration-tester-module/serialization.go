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

// SerializableStringConvRequest is a serializable string request.
type SerializableStringConvRequest struct {
	Style   pam.Style
	Request string
}

// SerializableStringConvResponse is a serializable string response.
type SerializableStringConvResponse struct {
	Style    pam.Style
	Response string
}

// SerializableBinaryConvRequest is a serializable binary request.
type SerializableBinaryConvRequest struct {
	Request []byte
}

// SerializableBinaryConvResponse is a serializable binary response.
type SerializableBinaryConvResponse struct {
	Response []byte
}

func init() {
	gob.Register(map[string]string{})
	gob.Register(Request{})
	gob.Register(pam.Item(0))
	gob.Register(pam.Error(0))
	gob.Register(pam.Style(0))
	gob.Register([]pam.ConvResponse{})
	gob.RegisterName("main.SerializableTransactionError",
		SerializableTransactionError{})
	gob.RegisterName("main.SerializableStringConvRequest",
		SerializableStringConvRequest{})
	gob.RegisterName("main.SerializableStringConvResponse",
		SerializableStringConvResponse{})
	gob.RegisterName("main.SerializableBinaryConvRequest",
		SerializableBinaryConvRequest{})
	gob.RegisterName("main.SerializableBinaryConvResponse",
		SerializableBinaryConvResponse{})
	gob.Register(utils.SerializableError{})
}
