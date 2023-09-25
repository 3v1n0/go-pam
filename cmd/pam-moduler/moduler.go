// pam-moduler is a tool to automate the creation of PAM Modules in go
//
// The file is created in the same package and directory as the package that
// creates the module
//
// The module implementation should define a pamModuleHandler object that
// implements the pam.ModuleHandler type and that will be used for each callback
//
// Otherwise it's possible to provide a typename from command line that will
// be used for this purpose
//
// For example:
//
//  //go:generate go run github.com/msteinert/pam/pam-moduler
//  //go:generate go generate --skip="pam_module"
//  package main
//
//  import "github.com/msteinert/pam/v2"
//
//  type ExampleHandler struct{}
//  var pamModuleHandler pam.ModuleHandler = &ExampleHandler{}
//
//  func (h *ExampleHandler) AcctMgmt(pam.ModuleTransaction, pam.Flags, []string) error {
//  	return nil
//  }
//
//  func (h *ExampleHandler) Authenticate(pam.ModuleTransaction, pam.Flags, []string) error {
//  	return nil
//  }
//
//  func (h *ExampleHandler) ChangeAuthTok(pam.ModuleTransaction, pam.Flags, []string) error {
//  	return nil
//  }
//
//  func (h *ExampleHandler) OpenSession(pam.ModuleTransaction, pam.Flags, []string) error {
//  	return nil
//  }
//
//  func (h *ExampleHandler) CloseSession(pam.ModuleTransaction, pam.Flags, []string) error {
//  	return nil
//  }
//
//  func (h *ExampleHandler) SetCred(pam.ModuleTransaction, pam.Flags, []string) error {
//  	return nil
//  }

// Package main provides the module shared library.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const toolName = "pam-moduler"

var (
	output           = flag.String("output", "", "output file name; default srcdir/pam_module.go")
	libName          = flag.String("libname", "", "output library name; default pam_go.so")
	typeName         = flag.String("type", "", "type name to be used as pam.ModuleHandler")
	buildTags        = flag.String("tags", "", "build tags expression to append to use in the go:build directive")
	skipGenerator    = flag.Bool("no-generator", false, "whether to add go:generate directives to the generated source")
	moduleBuildFlags = flag.String("build-flags", "", "comma-separated list of go build flags to use when generating the module")
	moduleBuildTags  = flag.String("build-tags", "", "comma-separated list of build tags to use when generating the module")
	noMain           = flag.Bool("no-main", false, "whether to add an empty main to generated file")
)

// Usage is a replacement usage function for the flags package.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", toolName)
	fmt.Fprintf(os.Stderr, "\t%s [flags] [-output O] [-libname pam_go] [-type N]\n", toolName)
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix(toolName + ": ")
	flag.Usage = Usage
	flag.Parse()

	if *skipGenerator {
		if *libName != "" {
			fmt.Fprintf(os.Stderr,
				"Generator directives disabled, libname will have no effect\n")
		}
		if *moduleBuildTags != "" {
			fmt.Fprintf(os.Stderr,
				"Generator directives disabled, build-tags will have no effect\n")
		}
		if *moduleBuildFlags != "" {
			fmt.Fprintf(os.Stderr,
				"Generator directives disabled, build-flags will have no effect\n")
		}
	}

	lib := *libName
	if lib == "" {
		lib = "pam_go"
	} else {
		lib, _ = strings.CutSuffix(lib, ".so")
		lib, _ = strings.CutPrefix(lib, "lib")
	}

	outputName, _ := strings.CutSuffix(*output, ".go")
	if outputName == "" {
		baseName := "pam_module"
		outputName = filepath.Join(".", strings.ToLower(baseName))
	}
	outputName = outputName + ".go"

	var tags string
	if *buildTags != "" {
		tags = *buildTags
	}

	var generateTags []string
	if len(*moduleBuildTags) > 0 {
		generateTags = strings.Split(*moduleBuildTags, ",")
	}

	var buildFlags []string
	if *moduleBuildFlags != "" {
		buildFlags = strings.Split(*moduleBuildFlags, ",")
	}

	g := Generator{
		outputName:   outputName,
		libName:      lib,
		tags:         tags,
		skipGenerate: *skipGenerator,
		buildFlags:   buildFlags,
		generateTags: generateTags,
		noMain:       *noMain,
		typeName:     *typeName,
	}

	// Print the header and package clause.
	g.printf("// Code generated by \"%s %s\"; DO NOT EDIT.\n",
		toolName, strings.Join(os.Args[1:], " "))
	g.printf("\n")

	// Generate the code
	g.generate()

	// Format the output.
	src := g.format()

	// Write to file.
	err := os.WriteFile(outputName, src, 0600)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	buf bytes.Buffer // Accumulated output.

	libName      string
	outputName   string
	typeName     string
	tags         string
	generateTags []string
	buildFlags   []string
	skipGenerate bool
	noMain       bool
}

func (g *Generator) printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

// generate produces the String method for the named type.
func (g *Generator) generate() {
	if g.tags != "" {
		g.printf("//go:build %s\n", g.tags)
	}

	var buildTagsArg string
	if len(g.generateTags) > 0 {
		buildTagsArg = fmt.Sprintf("-tags %s", strings.Join(g.generateTags, ","))
	}

	// We use a slice since we want to keep order, for reproducible builds.
	vFuncs := []struct {
		cName  string
		goName string
	}{
		{"authenticate", "Authenticate"},
		{"setcred", "SetCred"},
		{"acct_mgmt", "AcctMgmt"},
		{"open_session", "OpenSession"},
		{"close_session", "CloseSession"},
		{"chauthtok", "ChangeAuthTok"},
	}

	if !g.skipGenerate {
		g.printf(`//go:generate go build "-ldflags=-extldflags -Wl,-soname,%[1]s.so" `+
			`-buildmode=c-shared -o %[1]s.so %[2]s %[3]s
	`,
			g.libName, buildTagsArg, strings.Join(g.buildFlags, " "))
	}

	g.printf(`
// Package main is the package for the PAM module library.
package main

/*
#cgo LDFLAGS: -lpam -fPIC
#include <security/pam_modules.h>

typedef const char _const_char_t;
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"unsafe"
	"github.com/msteinert/pam/v2"
)
`)

	if g.typeName != "" {
		g.printf(`
var pamModuleHandler pam.ModuleHandler = &%[1]s{}
`, g.typeName)
	} else {
		g.printf(`
// Do a typecheck at compile time
var _ pam.ModuleHandler = pamModuleHandler;
`)
	}

	g.printf(`
// sliceFromArgv returns a slice of strings given to the PAM module.
func sliceFromArgv(argc C.int, argv **C._const_char_t) []string {
	r := make([]string, 0, argc)
	for _, s := range unsafe.Slice(argv, argc) {
		r = append(r, C.GoString(s))
	}
	return r
}

// handlePamCall is the function that translates C pam requests to Go.
func handlePamCall(pamh *C.pam_handle_t, flags C.int, argc C.int,
	argv **C._const_char_t, moduleFunc pam.ModuleHandlerFunc) C.int {
	if pamModuleHandler == nil {
		return C.int(pam.ErrNoModuleData)
	}

	if moduleFunc == nil {
		return C.int(pam.ErrIgnore)
	}

	err := moduleFunc(pam.NewModuleTransaction(pam.NativeHandle(pamh)),
		pam.Flags(flags), sliceFromArgv(argc, argv))

	if err == nil {
		return 0;
	}

	if (pam.Flags(flags) & pam.Silent) == 0 {
		fmt.Fprintf(os.Stderr, "module returned error: %%v\n", err)
	}

	var pamErr pam.Error
	if errors.As(err, &pamErr) {
		return C.int(pamErr)
	}

	return C.int(pam.ErrSystem)
}
`)

	for _, f := range vFuncs {
		g.printf(`
//export pam_sm_%[1]s
func pam_sm_%[1]s(pamh *C.pam_handle_t, flags C.int, argc C.int, argv **C._const_char_t) C.int {
	return handlePamCall(pamh, flags, argc, argv, pamModuleHandler.%[2]s)
}
`, f.cName, f.goName)
	}

	if !g.noMain {
		g.printf("\nfunc main() {}\n")
	}
}

// format returns the gofmt-ed contents of the Generator's buffer.
func (g *Generator) format() []byte {
	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		// Should never happen, but can arise when developing this code.
		// The user can compile the output to see the error.
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		return g.buf.Bytes()
	}
	return src
}
