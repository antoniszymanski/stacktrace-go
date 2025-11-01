// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package stacktrace

import (
	"io"
	"runtime"
	"testing"
)

var fn func()

func init() {
	fn = func() { (*runtime.Func)(nil).Entry() }
	for range 16 {
		oldFn := fn
		fn = func() { oldFn() }
	}
	oldFn := fn
	fn = func() {
		defer Handle(false, nil, nil)
		oldFn()
	}
	output = io.Discard
}

func BenchmarkHandle(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		fn()
	}
}
