// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-FileCopyrightText: 2018-2019 Elasticsearch B.V.
// SPDX-FileCopyrightText: 2009 The Go Authors
// SPDX-License-Identifier: MPL-2.0

package stacktrace

import (
	"iter"
	"runtime"
	"strings"
	"unsafe"

	"github.com/antoniszymanski/gopc-go"
)

func CallStack(skip int, predicate func(frame runtime.Frame) bool) iter.Seq[runtime.Frame] {
	pcs := make([]uintptr, 16)
	for {
		n := callers(skip+1, pcs)
		if n < len(pcs) {
			pcs = pcs[:n]
			break
		}
		pcs = make([]uintptr, 2*len(pcs))
	}
	if gopc := gopc.Get(); gopc != 0 {
		pcs = append(pcs, gopc)
	}
	frames := runtime.CallersFrames(pcs)
	return func(yield func(runtime.Frame) bool) {
		for {
			frame, more := frames.Next()
			if isUnexportedRuntime(frame.Function) ||
				frame.Function == "github.com/antoniszymanski/stacktrace-go.Go.func1" ||
				(predicate != nil && !predicate(frame)) {
				goto skip
			}
			if !yield(frame) {
				return
			}
		skip:
			if !more {
				return
			}
		}
	}
}

//go:linkname callers runtime.callers
func callers(skip int, pcs []uintptr) int

// isUnexportedRuntime reports whether name is an unexported runtime function.
//
// https://github.com/golang/go/blob/release-branch.go1.25/src/runtime/traceback.go#L1166
func isUnexportedRuntime(name string) bool {
	// Check and remove package qualifier.
	name, found := strings.CutPrefix(name, "runtime.")
	if !found {
		return false
	}
	rcvr := ""

	// Extract receiver type, if any.
	// For example, runtime.(*Func).Entry
	i := len(name) - 1
	for i >= 0 && name[i] != '.' {
		i--
	}
	if i >= 0 {
		rcvr = name[:i]
		name = name[i+1:]
		// Remove parentheses and star for pointer receivers.
		if len(rcvr) >= 3 && rcvr[0] == '(' && rcvr[1] == '*' && rcvr[len(rcvr)-1] == ')' {
			rcvr = rcvr[2 : len(rcvr)-1]
		}
	}

	// Unexported functions and unexported methods on unexported types.
	return len(name) == 0 || name[0] < 'A' || name[0] > 'Z' || (len(rcvr) > 0 && (rcvr[0] < 'A' || rcvr[0] > 'Z'))
}

// SplitFunctionPath splits the function path as formatted in
// [runtime.Frame.Function], and returns the package path and
// function name components.
func SplitFunctionPath(functionPath string) (packagePath string, functionName string) {
	if functionPath == "" {
		return "", ""
	}
	// The last part of a package path will always have "."
	// encoded as "%2e", so we can pick off the package path
	// by finding the last part of the package path, and then
	// the proceeding ".".
	//
	// Unexported method names may contain the package path.
	// In these cases, the method receiver will be enclosed
	// in parentheses, so we can treat that as the start of
	// the function name.
	if sep := strings.Index(functionPath, ".("); sep >= 0 {
		packagePath = unescape(functionPath[:sep+1])
		functionName = functionPath[sep+1:]
	} else {
		offset := 1
		if sep := strings.LastIndexByte(functionPath, '/'); sep >= 0 {
			offset += sep
		}
		if sep := strings.IndexByte(functionPath[offset:], '.'); sep >= 0 {
			packagePath = unescape(functionPath[:offset+sep+1])
			functionName = functionPath[offset+sep+1:]
		} else {
			functionName = functionPath // function path is invalid
		}
	}
	return packagePath, functionName
}

func unescape(s string) string {
	n := strings.Count(s, "%")
	if n == 0 {
		return s
	}
	dst := makeNoZero(len(s) - 2*n)[:0]
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '%' && i+2 < len(s) {
			b = unhex(s[i+1])<<4 | unhex(s[i+2])
			i += 2
		}
		dst = append(dst, b)
	}
	return unsafe.String(unsafe.SliceData(dst), len(dst))
}

//go:linkname makeNoZero internal/bytealg.MakeNoZero
func makeNoZero(length int) []byte

func unhex(b byte) byte {
	return 9*(b>>6) + (b & 15)
}
