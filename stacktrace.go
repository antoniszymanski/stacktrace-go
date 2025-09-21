// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-FileCopyrightText: 2018-2019 Elasticsearch B.V.
// SPDX-License-Identifier: MPL-2.0

package stacktrace

import (
	"iter"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
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
			const prefix = "runtime."
			switch {
			case len(frame.Function) >= len(prefix)+1 && frame.Function[:len(prefix)] == prefix:
				r, _ := utf8.DecodeRuneInString(frame.Function[len(prefix):])
				if !unicode.IsUpper(r) {
					goto skip
				}
			case frame.Function == "github.com/antoniszymanski/stacktrace-go.Go.func1":
				goto skip
			case predicate != nil && !predicate(frame):
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
			b = fromHex(s[i+1])<<4 | fromHex(s[i+2])
			i += 2
		}
		dst = append(dst, b)
	}
	return unsafe.String(unsafe.SliceData(dst), len(dst))
}

//go:linkname makeNoZero internal/bytealg.MakeNoZero
func makeNoZero(length int) []byte

func fromHex(b byte) byte {
	if b >= 'a' {
		return 10 + b - 'a'
	}
	return b - '0'
}
