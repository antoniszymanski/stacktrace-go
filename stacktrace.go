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

func callStack(skip int, predicate func(frame runtime.Frame) bool) iter.Seq[runtime.Frame] {
	callers := make([]uintptr, 16)
	for {
		n := runtime.Callers(2+skip, callers)
		if n < len(callers) {
			callers = callers[:n]
			break
		}
		callers = make([]uintptr, 2*len(callers))
	}
	if gopc := gopc.Get(); gopc != 0 {
		callers = append(callers, gopc)
	}
	frames := runtime.CallersFrames(callers)
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

// splitFuncPath splits the function path as formatted in
// [runtime.Frame.Function], and returns the package path and
// function name components.
func splitFuncPath(funcPath string) (pkgPath string, funcName string) {
	if funcPath == "" {
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
	if sep := strings.Index(funcPath, ".("); sep >= 0 {
		pkgPath = unescape(funcPath[:sep+1])
		funcName = funcPath[sep+1:]
	} else {
		offset := 1
		if sep := strings.LastIndexByte(funcPath, '/'); sep >= 0 {
			offset += sep
		}
		if sep := strings.IndexByte(funcPath[offset:], '.'); sep >= 0 {
			pkgPath = unescape(funcPath[:offset+sep+1])
			funcName = funcPath[offset+sep+1:]
		} else {
			funcName = funcPath // function path is invalid
		}
	}
	return pkgPath, funcName
}

func unescape(s string) string {
	var n int
	for _, b := range []byte(s) {
		if b == '%' {
			n++
		}
	}
	if n == 0 {
		return s
	}
	dst := make([]byte, 0, len(s)-2*n)
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

func fromHex(b byte) byte {
	if b >= 'a' {
		return 10 + b - 'a'
	}
	return b - '0'
}
