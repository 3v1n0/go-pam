/* Code generated by cmd/cgo; DO NOT EDIT. */

/* package github.com/msteinert/pam/cmd/pam-moduler/tests/debug-module */


#line 1 "cgo-builtin-export-prolog"

#include <stddef.h>

#ifndef GO_CGO_EXPORT_PROLOGUE_H
#define GO_CGO_EXPORT_PROLOGUE_H

#ifndef GO_CGO_GOSTRING_TYPEDEF
typedef struct { const char *p; ptrdiff_t n; } _GoString_;
#endif

#endif

/* Start of preamble from import "C" comments.  */


#line 8 "pam_module.go"


#include <security/pam_modules.h>

typedef const char _const_char_t;

#line 1 "cgo-generated-wrapper"


/* End of preamble from import "C" comments.  */


/* Start of boilerplate cgo prologue.  */
#line 1 "cgo-gcc-export-header-prolog"

#ifndef GO_CGO_PROLOGUE_H
#define GO_CGO_PROLOGUE_H

typedef signed char GoInt8;
typedef unsigned char GoUint8;
typedef short GoInt16;
typedef unsigned short GoUint16;
typedef int GoInt32;
typedef unsigned int GoUint32;
typedef long long GoInt64;
typedef unsigned long long GoUint64;
typedef GoInt64 GoInt;
typedef GoUint64 GoUint;
typedef size_t GoUintptr;
typedef float GoFloat32;
typedef double GoFloat64;
#ifdef _MSC_VER
#include <complex.h>
typedef _Fcomplex GoComplex64;
typedef _Dcomplex GoComplex128;
#else
typedef float _Complex GoComplex64;
typedef double _Complex GoComplex128;
#endif

/*
  static assertion to make sure the file is being used on architecture
  at least with matching size of GoInt.
*/
typedef char _check_for_64_bit_pointer_matching_GoInt[sizeof(void*)==64/8 ? 1:-1];

#ifndef GO_CGO_GOSTRING_TYPEDEF
typedef _GoString_ GoString;
#endif
typedef void *GoMap;
typedef void *GoChan;
typedef struct { void *t; void *v; } GoInterface;
typedef struct { void *data; GoInt len; GoInt cap; } GoSlice;

#endif

/* End of boilerplate cgo prologue.  */

#ifdef __cplusplus
extern "C" {
#endif

extern int pam_sm_authenticate(pam_handle_t* pamh, int flags, int argc, _const_char_t** argv);
extern int pam_sm_setcred(pam_handle_t* pamh, int flags, int argc, _const_char_t** argv);
extern int pam_sm_acct_mgmt(pam_handle_t* pamh, int flags, int argc, _const_char_t** argv);
extern int pam_sm_open_session(pam_handle_t* pamh, int flags, int argc, _const_char_t** argv);
extern int pam_sm_close_session(pam_handle_t* pamh, int flags, int argc, _const_char_t** argv);
extern int pam_sm_chauthtok(pam_handle_t* pamh, int flags, int argc, _const_char_t** argv);

#ifdef __cplusplus
}
#endif
