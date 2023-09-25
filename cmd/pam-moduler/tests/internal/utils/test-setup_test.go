package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isDir(t *testing.T, path string) bool {
	t.Helper()
	if file, err := os.Open(path); err == nil {
		if fileInfo, err := file.Stat(); err == nil {
			return fileInfo.IsDir()
		}
		t.Fatalf("error: %v", err)
	} else {
		t.Fatalf("error: %v", err)
	}
	return false
}

func Test_CreateTemporaryDir(t *testing.T) {
	t.Parallel()
	ts := NewTestSetup(t)
	dir := ts.CreateTemporaryDir("")
	if !isDir(t, dir) {
		t.Fatalf("%s not a directory", dir)
	}

	dir = ts.CreateTemporaryDir("foo-prefix-*")
	if !isDir(t, dir) {
		t.Fatalf("%s not a directory", dir)
	}
}

func Test_TestSetupWithWorkDir(t *testing.T) {
	t.Parallel()
	ts := NewTestSetup(t, WithWorkDir())
	if !isDir(t, ts.WorkDir()) {
		t.Fatalf("%s not a directory", ts.WorkDir())
	}
}

func Test_CreateService(t *testing.T) {
	t.Parallel()
	ts := NewTestSetup(t)

	tests := map[string]struct {
		services        []ServiceLine
		expectedContent string
	}{
		"empty":         {},
		"CApital-Empty": {},
		"auth-sufficient-permit": {
			services: []ServiceLine{
				{Auth, Sufficient, Permit.String(), []string{}},
			},
			expectedContent: "auth	sufficient	pam_permit.so",
		},
		"auth-sufficient-permit-args": {
			services: []ServiceLine{
				{Auth, Required, Deny.String(), []string{"a b c [d e]"}},
			},
			expectedContent: "auth	required	pam_deny.so	a b c [d e]",
		},
		"complete-custom": {
			services: []ServiceLine{
				{Account, Required, "pam_account_module.so", []string{"a", "b", "c", "[d e]"}},
				{Account, Required, Deny.String(), []string{}},
				{Auth, Requisite, "pam_auth_module.so", []string{}},
				{Auth, Requisite, Deny.String(), []string{}},
				{Password, Sufficient, "pam_password_module.so", []string{"arg"}},
				{Password, Sufficient, Deny.String(), []string{}},
				{Session, Optional, "pam_session_module.so", []string{""}},
				{Session, Optional, Deny.String(), []string{}},
			},
			expectedContent: `account	required	pam_account_module.so	a b c [d e]
account	required	pam_deny.so
auth	requisite	pam_auth_module.so
auth	requisite	pam_deny.so
password	sufficient	pam_password_module.so	arg
password	sufficient	pam_deny.so
session	optional	pam_session_module.so
session	optional	pam_deny.so`,
		},
	}

	for name, tc := range tests {
		tc := tc
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			service := ts.CreateService(name, tc.services)

			if filepath.Base(service) != strings.ToLower(name) {
				t.Fatalf("Invalid service name %s", service)
			}

			if bytes, err := os.ReadFile(service); err != nil {
				t.Fatalf("Failed reading %s: %v", service, err)
			} else {
				if string(bytes) != tc.expectedContent {
					t.Fatalf("Unexpected file content:\n%s\n---\n%s",
						tc.expectedContent, string(bytes))
				}
			}
		})
	}
}
