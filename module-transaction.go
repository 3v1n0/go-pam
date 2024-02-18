// Package pam provides a wrapper for the PAM application API.
package pam

/*
#include "transaction.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime/cgo"
	"unsafe"
)

const maxNumMsg = C.PAM_MAX_NUM_MSG

// ModuleTransaction is an interface that a pam module transaction
// should implement.
type ModuleTransaction interface {
	SetItem(Item, string) error
	GetItem(Item) (string, error)
	PutEnv(nameVal string) error
	GetEnv(name string) string
	GetEnvList() (map[string]string, error)
	GetUser(prompt string) (string, error)
	SetData(key string, data any) error
	GetData(key string) (any, error)
	StartStringConv(style Style, prompt string) (StringConvResponse, error)
	StartStringConvf(style Style, format string, args ...interface{}) (
		StringConvResponse, error)
	StartConv(ConvRequest) (ConvResponse, error)
	StartConvMulti([]ConvRequest) ([]ConvResponse, error)
}

// ModuleHandlerFunc is a function type used by the ModuleHandler.
type ModuleHandlerFunc func(ModuleTransaction, Flags, []string) error

// ModuleTransaction is the module-side handle for a PAM transaction.
type moduleTransaction struct {
	transactionBase
}

// ModuleHandler is an interface for objects that can be used to create
// PAM modules from go.
type ModuleHandler interface {
	AcctMgmt(ModuleTransaction, Flags, []string) error
	Authenticate(ModuleTransaction, Flags, []string) error
	ChangeAuthTok(ModuleTransaction, Flags, []string) error
	CloseSession(ModuleTransaction, Flags, []string) error
	OpenSession(ModuleTransaction, Flags, []string) error
	SetCred(ModuleTransaction, Flags, []string) error
}

// ModuleTransactionInvoker is an interface that a pam module transaction
// should implement to redirect requests from C handlers to go,
type ModuleTransactionInvoker interface {
	ModuleTransaction
	InvokeHandler(handler ModuleHandlerFunc, flags Flags, args []string) error
}

// NewModuleTransactionInvoker allows initializing a transaction invoker from
// the module side.
func NewModuleTransactionInvoker(handle NativeHandle) ModuleTransactionInvoker {
	return &moduleTransaction{transactionBase{handle: handle}}
}

func (m *moduleTransaction) InvokeHandler(handler ModuleHandlerFunc,
	flags Flags, args []string) error {
	invoker := func() error {
		if handler == nil {
			return ErrIgnore
		}
		err := handler(m, flags, args)
		if err != nil {
			service, _ := m.GetItem(Service)

			var pamErr Error
			if !errors.As(err, &pamErr) {
				err = ErrSystem
			}

			if pamErr == ErrIgnore || service == "" {
				return err
			}

			return fmt.Errorf("%s failed: %w", service, err)
		}
		return nil
	}
	err := invoker()
	if errors.Is(err, Error(0)) {
		err = nil
	}
	var status int32
	if err != nil {
		status = int32(ErrSystem)

		var pamErr Error
		if errors.As(err, &pamErr) {
			status = int32(pamErr)
		}
	}
	m.lastStatus.Store(status)
	return err
}

type moduleTransactionIface interface {
	getUser(outUser **C.char, prompt *C.char) C.int
	setData(key *C.char, handle C.uintptr_t) C.int
	getData(key *C.char, outHandle *C.uintptr_t) C.int
	getConv() (*C.struct_pam_conv, error)
	startConv(conv *C.struct_pam_conv, nMsg C.int,
		messages **C.struct_pam_message,
		outResponses **C.struct_pam_response) C.int
}

func (m *moduleTransaction) getUser(outUser **C.char, prompt *C.char) C.int {
	return C.pam_get_user(m.handle, outUser, prompt)
}

// getUserImpl is the default implementation for GetUser, but kept as private so
// that can be used to test the pam package
func (m *moduleTransaction) getUserImpl(iface moduleTransactionIface,
	prompt string) (string, error) {
	var user *C.char
	var cPrompt = C.CString(prompt)
	defer C.free(unsafe.Pointer(cPrompt))
	err := m.handlePamStatus(iface.getUser(&user, cPrompt))
	if err != nil {
		return "", err
	}
	return C.GoString(user), nil
}

// GetUser is similar to GetItem(User), but it would start a conversation if
// no user is currently set in PAM.
func (m *moduleTransaction) GetUser(prompt string) (string, error) {
	return m.getUserImpl(m, prompt)
}

// SetData allows to save any value in the module data that is preserved
// during the whole time the module is loaded.
func (m *moduleTransaction) SetData(key string, data any) error {
	return m.setDataImpl(m, key, data)
}

func (m *moduleTransaction) setData(key *C.char, handle C.uintptr_t) C.int {
	return C.set_data(m.handle, key, handle)
}

// setDataImpl is the implementation for SetData for testing purposes.
func (m *moduleTransaction) setDataImpl(iface moduleTransactionIface,
	key string, data any) error {
	var cKey = C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	var handle cgo.Handle
	if data != nil {
		handle = cgo.NewHandle(data)
	}
	return m.handlePamStatus(iface.setData(cKey, C.uintptr_t(handle)))
}

//export _go_pam_data_cleanup
func _go_pam_data_cleanup(h NativeHandle, handle C.uintptr_t, status C.int) {
	cgo.Handle(handle).Delete()
}

// GetData allows to get any value from the module data saved using SetData
// that is preserved across the whole time the module is loaded.
func (m *moduleTransaction) GetData(key string) (any, error) {
	return m.getDataImpl(m, key)
}

func (m *moduleTransaction) getData(key *C.char, outHandle *C.uintptr_t) C.int {
	return C.get_data(m.handle, key, outHandle)
}

// getDataImpl is the implementation for GetData for testing purposes.
func (m *moduleTransaction) getDataImpl(iface moduleTransactionIface,
	key string) (any, error) {
	var cKey = C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	var handle C.uintptr_t
	if err := m.handlePamStatus(iface.getData(cKey, &handle)); err != nil {
		return nil, err
	}
	if goHandle := cgo.Handle(handle); goHandle != cgo.Handle(0) {
		return goHandle.Value(), nil
	}

	return nil, m.handlePamStatus(C.int(ErrNoModuleData))
}

// getConv is a private function to get the conversation pointer to be used
// with C.do_conv() to initiate conversations.
func (m *moduleTransaction) getConv() (*C.struct_pam_conv, error) {
	var convPtr unsafe.Pointer

	if err := m.handlePamStatus(
		C.pam_get_item(m.handle, C.PAM_CONV, &convPtr)); err != nil {
		return nil, err
	}

	return (*C.struct_pam_conv)(convPtr), nil
}

// ConvRequest is an interface that all the Conversation requests should
// implement.
type ConvRequest interface {
	Style() Style
}

// ConvResponse is an interface that all the Conversation responses should
// implement.
type ConvResponse interface {
	Style() Style
}

// StringConvRequest is a ConvRequest for performing text-based conversations.
type StringConvRequest struct {
	style  Style
	prompt string
}

// NewStringConvRequest creates a new StringConvRequest.
func NewStringConvRequest(style Style, prompt string) StringConvRequest {
	return StringConvRequest{style, prompt}
}

// Style returns the conversation style of the StringConvRequest.
func (s StringConvRequest) Style() Style {
	return s.style
}

// Prompt returns the conversation style of the StringConvRequest.
func (s StringConvRequest) Prompt() string {
	return s.prompt
}

// StringConvResponse is an interface that string Conversation responses implements.
type StringConvResponse interface {
	ConvResponse
	Response() string
}

// stringConvResponse is a StringConvResponse implementation used for text-based
// conversation responses.
type stringConvResponse struct {
	style    Style
	response string
}

// Style returns the conversation style of the StringConvResponse.
func (s stringConvResponse) Style() Style {
	return s.style
}

// Response returns the string response of the conversation.
func (s stringConvResponse) Response() string {
	return s.response
}

// StartStringConv starts a text-based conversation using the provided style
// and prompt.
func (m *moduleTransaction) StartStringConv(style Style, prompt string) (
	StringConvResponse, error) {
	return m.startStringConvImpl(m, style, prompt)
}

func (m *moduleTransaction) startStringConvImpl(iface moduleTransactionIface,
	style Style, prompt string) (
	StringConvResponse, error) {
	switch style {
	case BinaryPrompt:
		return nil, fmt.Errorf("%w: binary style is not supported", ErrConv)
	}

	res, err := m.startConvImpl(iface, NewStringConvRequest(style, prompt))
	if err != nil {
		return nil, err
	}

	stringRes, _ := res.(stringConvResponse)
	return stringRes, nil
}

// StartStringConvf allows to start string conversation with formatting support.
func (m *moduleTransaction) StartStringConvf(style Style, format string, args ...interface{}) (
	StringConvResponse, error) {
	return m.StartStringConv(style, fmt.Sprintf(format, args...))
}

// StartConv initiates a PAM conversation using the provided ConvRequest.
func (m *moduleTransaction) StartConv(req ConvRequest) (
	ConvResponse, error) {
	return m.startConvImpl(m, req)
}

func (m *moduleTransaction) startConvImpl(iface moduleTransactionIface, req ConvRequest) (
	ConvResponse, error) {
	resp, err := m.startConvMultiImpl(iface, []ConvRequest{req})
	if err != nil {
		return nil, err
	}
	if len(resp) != 1 {
		return nil, fmt.Errorf("%w: not enough values returned", ErrConv)
	}
	return resp[0], nil
}

func (m *moduleTransaction) startConv(conv *C.struct_pam_conv, nMsg C.int,
	messages **C.struct_pam_message, outResponses **C.struct_pam_response) C.int {
	return C.start_pam_conv(conv, nMsg, messages, outResponses)
}

// startConvMultiImpl is the implementation for GetData for testing purposes.
func (m *moduleTransaction) startConvMultiImpl(iface moduleTransactionIface,
	requests []ConvRequest) (responses []ConvResponse, err error) {
	defer func() {
		if err == nil {
			_ = m.handlePamStatus(success)
			return
		}
		var pamErr Error
		if !errors.As(err, &pamErr) {
			err = errors.Join(ErrConv, err)
			pamErr = ErrConv
		}
		_ = m.handlePamStatus(C.int(pamErr))
	}()

	if len(requests) == 0 {
		return nil, errors.New("no requests defined")
	}
	if len(requests) > maxNumMsg {
		return nil, errors.New("too many requests")
	}

	conv, err := iface.getConv()
	if err != nil {
		return nil, err
	}

	if conv == nil || conv.conv == nil {
		return nil, errors.New("impossible to find conv handler")
	}

	// FIXME: Just use make([]C.struct_pam_message, 0, len(requests))
	// and append, when it's possible to use runtime.Pinner
	var cMessagePtr *C.struct_pam_message
	cMessages := (**C.struct_pam_message)(C.calloc(C.size_t(len(requests)),
		(C.size_t)(unsafe.Sizeof(cMessagePtr))))
	defer C.free(unsafe.Pointer(cMessages))
	goMsgs := unsafe.Slice(cMessages, len(requests))

	for i, req := range requests {
		var cBytes unsafe.Pointer
		switch r := req.(type) {
		case StringConvRequest:
			cBytes = unsafe.Pointer(C.CString(r.Prompt()))
			defer C.free(cBytes)
		default:
			return nil, fmt.Errorf("unsupported conversation type %#v", r)
		}

		goMsgs[i] = &C.struct_pam_message{
			msg_style: C.int(req.Style()),
			msg:       (*C.char)(cBytes),
		}
	}

	var cResponses *C.struct_pam_response
	ret := iface.startConv(conv, C.int(len(requests)), cMessages, &cResponses)
	if ret != success {
		return nil, Error(ret)
	}

	goResponses := unsafe.Slice(cResponses, len(requests))
	defer func() {
		for _, resp := range goResponses {
			C.free(unsafe.Pointer(resp.resp))
		}
		C.free(unsafe.Pointer(cResponses))
	}()

	responses = make([]ConvResponse, 0, len(requests))
	for i, resp := range goResponses {
		msgStyle := requests[i].Style()
		switch msgStyle {
		case PromptEchoOff:
			fallthrough
		case PromptEchoOn:
			fallthrough
		case ErrorMsg:
			fallthrough
		case TextInfo:
			responses = append(responses, stringConvResponse{
				style:    msgStyle,
				response: C.GoString(resp.resp),
			})
		default:
			return nil,
				fmt.Errorf("unsupported conversation type %v", msgStyle)
		}
	}

	return responses, nil
}

// StartConvMulti initiates a PAM conversation with multiple ConvRequest's.
func (m *moduleTransaction) StartConvMulti(requests []ConvRequest) (
	[]ConvResponse, error) {
	return m.startConvMultiImpl(m, requests)
}
