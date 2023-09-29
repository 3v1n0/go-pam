// Package pam provides a wrapper for the PAM application API.
package pam

//#cgo CFLAGS: -Wall -std=c99
//#cgo LDFLAGS: -lpam
//
//#include <security/pam_appl.h>
//#include <stdlib.h>
//#include <stdint.h>
//
//#ifdef PAM_BINARY_PROMPT
//#define BINARY_PROMPT_IS_SUPPORTED 1
//#else
//#include <limits.h>
//#define PAM_BINARY_PROMPT INT_MAX
//#define BINARY_PROMPT_IS_SUPPORTED 0
//#endif
//
//void init_pam_conv(struct pam_conv *conv, uintptr_t);
//int pam_start_confdir(const char *service_name, const char *user, const struct pam_conv *pam_conversation, const char *confdir, pam_handle_t **pamh) __attribute__ ((weak));
//int check_pam_start_confdir(void);
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"runtime/cgo"
	"strings"
	"sync/atomic"
	"unsafe"
)

// Type for PAM Return types
type ReturnType int

// Pam Return types
const (
	// Successful function return
	Success ReturnType = C.PAM_SUCCESS
	// dlopen() failure when dynamically loading a service module
	OpenErr ReturnType = C.PAM_OPEN_ERR
	// Symbol not found
	SymbolErr ReturnType = C.PAM_SYMBOL_ERR
	// Error in service module
	ServiceErr ReturnType = C.PAM_SERVICE_ERR
	// System error
	SystemErr ReturnType = C.PAM_SYSTEM_ERR
	// Memory buffer error
	BufErr ReturnType = C.PAM_BUF_ERR
	// Permission denied
	PermDenied ReturnType = C.PAM_PERM_DENIED
	// Authentication failure
	AuthErr ReturnType = C.PAM_AUTH_ERR
	// Can not access authentication data due to insufficient credentials
	CredInsufficient ReturnType = C.PAM_CRED_INSUFFICIENT
	// Underlying authentication service can not retrieve authentication
	// information
	AuthinfoUnavail ReturnType = C.PAM_AUTHINFO_UNAVAIL
	// User not known to the underlying authentication module
	UserUnknown ReturnType = C.PAM_USER_UNKNOWN
	// An authentication service has maintained a retry count which has been
	// reached.
	// No further retries should be attempted
	Maxtries ReturnType = C.PAM_MAXTRIES
	// New authentication token required. This is normally returned if the
	// machine security policies require that the password should be changed
	// because the password is nil or it has aged
	NewAuthtokReqd ReturnType = C.PAM_NEW_AUTHTOK_REQD
	// User account has expired
	AcctExpired ReturnType = C.PAM_ACCT_EXPIRED
	// Can not make/remove an entry for the specified session
	SessionErr ReturnType = C.PAM_SESSION_ERR
	// Underlying authentication service can not retrieve user credentials
	CredUnavail ReturnType = C.PAM_CRED_UNAVAIL
	// User credentials expired
	CredExpired ReturnType = C.PAM_CRED_EXPIRED
	// Failure setting user credentials
	CredErr ReturnType = C.PAM_CRED_ERR
	// No module specific data is present
	NoModuleData ReturnType = C.PAM_NO_MODULE_DATA
	// Conversation error
	ConvErr ReturnType = C.PAM_CONV_ERR
	// Authentication token manipulation error
	AuthtokErr ReturnType = C.PAM_AUTHTOK_ERR
	// Authentication information cannot be recovered
	AuthtokRecoveryErr ReturnType = C.PAM_AUTHTOK_RECOVERY_ERR
	// Authentication token lock busy
	AuthtokLockBusy ReturnType = C.PAM_AUTHTOK_LOCK_BUSY
	// Authentication token aging disabled
	AuthtokDisableAging ReturnType = C.PAM_AUTHTOK_DISABLE_AGING
	// Preliminary check by password service
	TryAgain ReturnType = C.PAM_TRY_AGAIN
	// Ignore underlying account module regardless of whether the control flag
	// is required, optional, or sufficient
	Ignore ReturnType = C.PAM_IGNORE
	// Critical error (?module fail now request)
	Abort ReturnType = C.PAM_ABORT
	// user's authentication token has expired
	AuthtokExpired ReturnType = C.PAM_AUTHTOK_EXPIRED
	// module is not known
	ModuleUnknown ReturnType = C.PAM_MODULE_UNKNOWN
	// Bad item passed to pam_*_item()
	BadItem ReturnType = C.PAM_BAD_ITEM
	// conversation function is event driven and data is not available yet
	ConvAgain ReturnType = C.PAM_CONV_AGAIN
	// please call this function again to complete authentication stack.
	// Before calling again, verify that conversation is completed
	Incomplete ReturnType = C.PAM_INCOMPLETE
)

func (rt ReturnType) Error() string {
	return fmt.Sprintf("%d: %s", rt, C.GoString(C.pam_strerror(nil, C.int(rt))))
}

func (rt ReturnType) toC() C.int {
	return C.int(rt)
}

// Style is the type of message that the conversation handler should display.
type Style int

// Coversation handler style types.
const (
	// PromptEchoOff indicates the conversation handler should obtain a
	// string without echoing any text.
	PromptEchoOff Style = C.PAM_PROMPT_ECHO_OFF
	// PromptEchoOn indicates the conversation handler should obtain a
	// string while echoing text.
	PromptEchoOn = C.PAM_PROMPT_ECHO_ON
	// ErrorMsg indicates the conversation handler should display an
	// error message.
	ErrorMsg = C.PAM_ERROR_MSG
	// TextInfo indicates the conversation handler should display some
	// text.
	TextInfo = C.PAM_TEXT_INFO
	// BinaryPrompt indicates the conversation handler that should implement
	// the private binary protocol
	BinaryPrompt = C.PAM_BINARY_PROMPT
)

// ConversationHandler is an interface for objects that can be used as
// conversation callbacks during PAM authentication.
type ConversationHandler interface {
	// RespondPAM receives a message style and a message string. If the
	// message Style is PromptEchoOff or PromptEchoOn then the function
	// should return a response string.
	RespondPAM(Style, string) (string, error)
}

// BinaryPointer exposes the type used for the data in a binary conversation
// it represents a pointer to data that is produced by the module and that
// must be parsed depending on the protocol in use
type BinaryPointer unsafe.Pointer

// BinaryConversationHandler is an interface for objects that can be used as
// conversation callbacks during PAM authentication if binary protocol is going
// to be supported.
type BinaryConversationHandler interface {
	ConversationHandler
	// RespondPAMBinary receives a pointer to the binary message. It's up to
	// the receiver to parse it according to the protocol specifications.
	// The function can return a byte array that will passed as pointer back
	// to the module.
	RespondPAMBinary(BinaryPointer) ([]byte, error)
}

// ConversationFunc is an adapter to allow the use of ordinary functions as
// conversation callbacks.
type ConversationFunc func(Style, string) (string, error)

// RespondPAM is a conversation callback adapter.
func (f ConversationFunc) RespondPAM(s Style, msg string) (string, error) {
	return f(s, msg)
}

// cbPAMConv is a wrapper for the conversation callback function.
//
//export cbPAMConv
func cbPAMConv(s C.int, msg *C.char, c C.uintptr_t) (*C.char, C.int) {
	var r string
	var err error
	v := cgo.Handle(c).Value()
	style := Style(s)
	switch cb := v.(type) {
	case BinaryConversationHandler:
		if style == BinaryPrompt {
			bytes, err := cb.RespondPAMBinary(BinaryPointer(msg))
			if err != nil {
				return nil, ConvAgain.toC()
			}
			return (*C.char)(C.CBytes(bytes)), Success.toC()
		} else {
			r, err = cb.RespondPAM(style, C.GoString(msg))
		}
	case ConversationHandler:
		if style == BinaryPrompt {
			return nil, AuthinfoUnavail.toC()
		}
		r, err = cb.RespondPAM(style, C.GoString(msg))
	}
	if err != nil {
		return nil, ConvErr.toC()
	}
	return C.CString(r), Success.toC()
}

// Transaction is the application's handle for a PAM transaction.
type Transaction struct {
	handle *C.pam_handle_t
	conv   *C.struct_pam_conv
	status int32
	c      cgo.Handle
}

// transactionFinalizer cleans up the PAM handle and deletes the callback
// function.
func transactionFinalizer(t *Transaction) {
	C.pam_end(t.handle, C.int(atomic.LoadInt32(&t.status)))
	t.c.Delete()
}

// Start initiates a new PAM transaction. Service is treated identically to
// how pam_start treats it internally.
//
// All application calls to PAM begin with Start*. The returned
// transaction provides an interface to the remainder of the API.
func Start(service, user string, handler ConversationHandler) (*Transaction, error) {
	return start(service, user, handler, "")
}

// StartFunc registers the handler func as a conversation handler.
func StartFunc(service, user string, handler func(Style, string) (string, error)) (*Transaction, error) {
	return Start(service, user, ConversationFunc(handler))
}

// StartConfDir initiates a new PAM transaction. Service is treated identically to
// how pam_start treats it internally.
// confdir allows to define where all pam services are defined. This is used to provide
// custom paths for tests.
//
// All application calls to PAM begin with Start*. The returned
// transaction provides an interface to the remainder of the API.
func StartConfDir(service, user string, handler ConversationHandler, confDir string) (*Transaction, error) {
	if !CheckPamHasStartConfdir() {
		return nil, &TransactionError{
			errors.New("StartConfDir() was used, but the pam version on the system is not recent enough"),
			SystemErr,
		}
	}

	return start(service, user, handler, confDir)
}

func start(service, user string, handler ConversationHandler, confDir string) (*Transaction, error) {
	switch handler.(type) {
	case BinaryConversationHandler:
		if !CheckPamHasBinaryProtocol() {
			return nil, &TransactionError{
				errors.New("BinaryConversationHandler() was used, but it is not supported by this platform"),
				SystemErr,
			}
		}
	}
	t := &Transaction{
		conv: &C.struct_pam_conv{},
		c:    cgo.NewHandle(handler),
	}
	C.init_pam_conv(t.conv, C.uintptr_t(t.c))
	runtime.SetFinalizer(t, transactionFinalizer)
	s := C.CString(service)
	defer C.free(unsafe.Pointer(s))
	var u *C.char
	if len(user) != 0 {
		u = C.CString(user)
		defer C.free(unsafe.Pointer(u))
	}
	var status C.int
	if confDir == "" {
		status = C.pam_start(s, u, t.conv, &t.handle)
	} else {
		c := C.CString(confDir)
		defer C.free(unsafe.Pointer(c))
		status = C.pam_start_confdir(s, u, t.conv, c, &t.handle)
	}
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return nil, &TransactionError{t, ReturnType(status)}
	}
	return t, nil
}

// transactionError is a private interface that is implemented by both
// TransactionError and Transaction
type transactionError interface {
	error
	Status() ReturnType
}

// TransactionError extends error to provide more detailed information
type TransactionError struct {
	error
	status ReturnType
}

// Status exposes the ReturnType for the error
func (e *TransactionError) Status() ReturnType {
	return e.status
}

// Error pretty prints the error from the status message
func (e *TransactionError) Error() string {
	return errors.Join(e.error, ReturnType(e.status)).Error()
}

func (t *Transaction) Error() string {
	return t.Status().Error()
}

// Status exposes the ReturnType for the last operation, as per its nature
// this value is not thread-safe and so if multiple goroutines are acting
// on the same transaction this should not be used, but one should rely on
// each operation return status.
func (t *Transaction) Status() ReturnType {
	return ReturnType(atomic.LoadInt32(&t.status))
}

// Item is a an PAM information type.
type Item int

// PAM Item types.
const (
	// Service is the name which identifies the PAM stack.
	Service Item = C.PAM_SERVICE
	// User identifies the username identity used by a service.
	User = C.PAM_USER
	// Tty is the terminal name.
	Tty = C.PAM_TTY
	// Rhost is the requesting host name.
	Rhost = C.PAM_RHOST
	// Authtok is the currently active authentication token.
	Authtok = C.PAM_AUTHTOK
	// Oldauthtok is the old authentication token.
	Oldauthtok = C.PAM_OLDAUTHTOK
	// Ruser is the requesting user name.
	Ruser = C.PAM_RUSER
	// UserPrompt is the string use to prompt for a username.
	UserPrompt = C.PAM_USER_PROMPT
)

// SetItem sets a PAM information item.
func (t *Transaction) SetItem(i Item, item string) error {
	cs := unsafe.Pointer(C.CString(item))
	defer C.free(cs)
	status := C.pam_set_item(t.handle, C.int(i), cs)
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return t
	}
	return nil
}

// GetItem retrieves a PAM information item.
func (t *Transaction) GetItem(i Item) (string, error) {
	var s unsafe.Pointer
	status := C.pam_get_item(t.handle, C.int(i), &s)
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return "", t
	}
	return C.GoString((*C.char)(s)), nil
}

// Flags are inputs to various PAM functions than be combined with a bitwise
// or. Refer to the official PAM documentation for which flags are accepted
// by which functions.
type Flags int

// PAM Flag types.
const (
	// Silent indicates that no messages should be emitted.
	Silent Flags = C.PAM_SILENT
	// DisallowNullAuthtok indicates that authorization should fail
	// if the user does not have a registered authentication token.
	DisallowNullAuthtok = C.PAM_DISALLOW_NULL_AUTHTOK
	// EstablishCred indicates that credentials should be established
	// for the user.
	EstablishCred = C.PAM_ESTABLISH_CRED
	// DeleteCred inidicates that credentials should be deleted.
	DeleteCred = C.PAM_DELETE_CRED
	// ReinitializeCred indicates that credentials should be fully
	// reinitialized.
	ReinitializeCred = C.PAM_REINITIALIZE_CRED
	// RefreshCred indicates that the lifetime of existing credentials
	// should be extended.
	RefreshCred = C.PAM_REFRESH_CRED
	// ChangeExpiredAuthtok indicates that the authentication token
	// should be changed if it has expired.
	ChangeExpiredAuthtok = C.PAM_CHANGE_EXPIRED_AUTHTOK
)

// Authenticate is used to authenticate the user.
//
// Valid flags: Silent, DisallowNullAuthtok
func (t *Transaction) Authenticate(f Flags) error {
	status := C.pam_authenticate(t.handle, C.int(f))
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return t
	}
	return nil
}

// SetCred is used to establish, maintain and delete the credentials of a
// user.
//
// Valid flags: EstablishCred, DeleteCred, ReinitializeCred, RefreshCred
func (t *Transaction) SetCred(f Flags) error {
	status := C.pam_setcred(t.handle, C.int(f))
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return t
	}
	return nil
}

// AcctMgmt is used to determine if the user's account is valid.
//
// Valid flags: Silent, DisallowNullAuthtok
func (t *Transaction) AcctMgmt(f Flags) error {
	status := C.pam_acct_mgmt(t.handle, C.int(f))
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return t
	}
	return nil
}

// ChangeAuthTok is used to change the authentication token.
//
// Valid flags: Silent, ChangeExpiredAuthtok
func (t *Transaction) ChangeAuthTok(f Flags) error {
	status := C.pam_chauthtok(t.handle, C.int(f))
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return t
	}
	return nil
}

// OpenSession sets up a user session for an authenticated user.
//
// Valid flags: Slient
func (t *Transaction) OpenSession(f Flags) error {
	status := C.pam_open_session(t.handle, C.int(f))
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return t
	}
	return nil
}

// CloseSession closes a previously opened session.
//
// Valid flags: Silent
func (t *Transaction) CloseSession(f Flags) error {
	status := C.pam_close_session(t.handle, C.int(f))
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return t
	}
	return nil
}

// PutEnv adds or changes the value of PAM environment variables.
//
// NAME=value will set a variable to a value.
// NAME= will set a variable to an empty value.
// NAME (without an "=") will delete a variable.
func (t *Transaction) PutEnv(nameval string) error {
	cs := C.CString(nameval)
	defer C.free(unsafe.Pointer(cs))
	status := C.pam_putenv(t.handle, cs)
	atomic.StoreInt32(&t.status, int32(status))
	if status != Success.toC() {
		return t
	}
	return nil
}

// GetEnv is used to retrieve a PAM environment variable.
func (t *Transaction) GetEnv(name string) string {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	value := C.pam_getenv(t.handle, cs)
	if value == nil {
		return ""
	}
	return C.GoString(value)
}

func next(p **C.char) **C.char {
	return (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + unsafe.Sizeof(p)))
}

// GetEnvList returns a copy of the PAM environment as a map.
func (t *Transaction) GetEnvList() (map[string]string, error) {
	env := make(map[string]string)
	p := C.pam_getenvlist(t.handle)
	if p == nil {
		atomic.StoreInt32(&t.status, C.PAM_BUF_ERR)
		return nil, t
	}
	for q := p; *q != nil; q = next(q) {
		chunks := strings.SplitN(C.GoString(*q), "=", 2)
		if len(chunks) == 2 {
			env[chunks[0]] = chunks[1]
		}
		C.free(unsafe.Pointer(*q))
	}
	C.free(unsafe.Pointer(p))
	return env, nil
}

// CheckPamHasStartConfdir return if pam on system supports pam_system_confdir
func CheckPamHasStartConfdir() bool {
	return C.check_pam_start_confdir() == 0
}

// CheckPamHasBinaryProtocol return if pam on system supports PAM_BINARY_PROMPT
func CheckPamHasBinaryProtocol() bool {
	return C.BINARY_PROMPT_IS_SUPPORTED != 0
}
