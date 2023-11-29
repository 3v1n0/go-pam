//go:build !go_pam_module

package pam

/*
#cgo CFLAGS: -Wall -std=c99
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"testing"
	"unsafe"
)

type mockModuleTransactionExpectations struct {
	UserPrompt string
}

type mockModuleTransactionReturnedData struct {
	User            string
	InteractiveUser bool
	Status          Error
}

type mockModuleTransaction struct {
	moduleTransaction
	T                   *testing.T
	Expectations        mockModuleTransactionExpectations
	RetData             mockModuleTransactionReturnedData
	ConversationHandler ConversationHandler
	allocatedData       []unsafe.Pointer
}

func newMockModuleTransaction(m *mockModuleTransaction) *mockModuleTransaction {
	runtime.SetFinalizer(m, func(m *mockModuleTransaction) {
		for _, ptr := range m.allocatedData {
			C.free(ptr)
		}
	})
	return m
}

func (m *mockModuleTransaction) getUser(outUser **C.char, prompt *C.char) C.int {
	goPrompt := C.GoString(prompt)
	if goPrompt != m.Expectations.UserPrompt {
		m.T.Fatalf("unexpected prompt: %s vs %s", goPrompt, m.Expectations.UserPrompt)
		return C.int(ErrAbort)
	}

	user := m.RetData.User
	if m.RetData.InteractiveUser || (m.RetData.User == "" && m.ConversationHandler != nil) {
		if m.ConversationHandler == nil {
			m.T.Fatalf("no conversation handler provided")
		}
		u, err := m.ConversationHandler.RespondPAM(PromptEchoOn, goPrompt)
		user = u

		if err != nil {
			ret := ErrAbort
			var txErr TransactionError
			if errors.As(err, &txErr) {
				ret = txErr.Status()
			}
			return C.int(ret)
		}
	}

	cUser := C.CString(user)
	m.allocatedData = append(m.allocatedData, unsafe.Pointer(cUser))

	*outUser = cUser
	return C.int(m.RetData.Status)
}

type mockConversationHandler struct {
	User              string
	ExpectedMessage   string
	CheckEmptyMessage bool
	ExpectedStyle     Style
	CheckZeroStyle    bool
}

func (c mockConversationHandler) RespondPAM(s Style, msg string) (string, error) {
	if (c.ExpectedMessage != "" || c.CheckEmptyMessage) &&
		msg != c.ExpectedMessage {
		return "", NewTransactionError(ErrConv,
			fmt.Errorf("unexpected prompt: %s vs %s", msg, c.ExpectedMessage))
	}

	if (c.ExpectedStyle != 0 || c.CheckZeroStyle) &&
		s != c.ExpectedStyle {
		return "", NewTransactionError(ErrConv,
			fmt.Errorf("unexpected style: %#v vs %#v", s, c.ExpectedStyle))
	}

	switch s {
	case PromptEchoOn:
		return c.User, nil
	}

	return "", NewTransactionError(ErrConv,
		fmt.Errorf("unhandled style: %v", s))
}
