// These go:generate directive allow to generate the module by just using
// `go generate` once in the module directory.
// This is not strictly needed

//go:generate go run github.com/msteinert/pam/cmd/pam-moduler
//go:generate go generate --skip="pam_module.go"

// Package main provides the module shared library.
package main

import (
	"fmt"

	"github.com/msteinert/pam"
)

type exampleHandler struct{}

var pamModuleHandler pam.ModuleHandler = &exampleHandler{}
var _ = pamModuleHandler

// AcctMgmt is the module handle function for account management.
func (h *exampleHandler) AcctMgmt(mt pam.ModuleTransaction, flags pam.Flags, args []string) error {
	return pam.NewTransactionError(pam.ErrIgnore,
		fmt.Errorf("AcctMgmt not implemented"))
}

// Authenticate is the module handle function for authentication.
func (h *exampleHandler) Authenticate(mt pam.ModuleTransaction, flags pam.Flags, args []string) error {
	return pam.ErrAuthinfoUnavail
}

// ChangeAuthTok is the module handle function for changing authentication token.
func (h *exampleHandler) ChangeAuthTok(mt pam.ModuleTransaction, flags pam.Flags, args []string) error {
	return pam.NewTransactionError(pam.ErrIgnore,
		fmt.Errorf("ChangeAuthTok not implemented"))
}

// OpenSession is the module handle function for open session.
func (h *exampleHandler) OpenSession(mt pam.ModuleTransaction, flags pam.Flags, args []string) error {
	return pam.NewTransactionError(pam.ErrIgnore,
		fmt.Errorf("OpenSession not implemented"))
}

// CloseSession is the module handle function for close session.
func (h *exampleHandler) CloseSession(mt pam.ModuleTransaction, flags pam.Flags, args []string) error {
	return pam.NewTransactionError(pam.ErrIgnore,
		fmt.Errorf("CloseSession not implemented"))
}

// SetCred is the module handle function for set credentials.
func (h *exampleHandler) SetCred(mt pam.ModuleTransaction, flags pam.Flags, args []string) error {
	return pam.NewTransactionError(pam.ErrIgnore,
		fmt.Errorf("SetCred not implemented"))
}
