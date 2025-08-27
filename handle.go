// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package stacktrace

import (
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"unsafe"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

var noColor bool
var output io.Writer
var mu sync.Mutex

func init() {
	Enable()
}

func Enable() {
	mu.Lock()
	defer mu.Unlock()
	noColor = os.Getenv("NO_COLOR") != "" ||
		(!isatty.IsTerminal(os.Stderr.Fd()) && !isatty.IsCygwinTerminal(os.Stderr.Fd()))
	if noColor {
		output = os.Stderr
	} else {
		output = colorable.NewColorableStderr()
	}
}

func Disable() {
	mu.Lock()
	defer mu.Unlock()
	output = nil
}

func Go(fn func(), print func(w io.Writer, r any)) {
	go func() {
		defer Handle(print, true)
		fn()
	}()
}

func Handle(print func(w io.Writer, r any), exit bool) {
	r := recover()
	if r == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if output == nil {
		return
	}
	if exit {
		defer os.Exit(2)
	}

	write3(bold+brightCyan, "panic: ", reset)
	if !noColor {
		write(bold + brightBlue)
	}
	if print != nil {
		print(output, r)
	} else {
		fmt.Fprint(output, r) //nolint:errcheck
	}
	if !noColor {
		write(reset)
	}
	write("\n")

	isFirst := true
	for frame := range callStack(2) {
		pkgPath, funcName := splitFuncPath(frame.Function)
		dir, name := path.Split(frame.File)
		offset := -1
		if frame.Func != nil {
			_, entry := frame.Func.FileLine(frame.Func.Entry())
			offset = frame.Line - entry
		}

		if isFirst {
			write3(red, "->", reset)
			write("  at ")
			write2(bold+brightYellow, pkgPath)
			write3(bold+brightGreen, funcName, reset)
			write("\n")
			write2(red, "->")
			write("       ")
			write2(bold+brightWhite, dir)
			write2(bold+brightCyan, name)
			write2(bold+brightGreen, ":")
			writeInt(frame.Line)
			writeOffset(bold+brightBlue, offset)
			write("\n\n")
		} else {
			write("    at ")
			write2(yellow, pkgPath)
			write2(brightGreen, funcName)
			write("\n         ")
			write2(brightWhite, dir)
			write2(brightCyan, name)
			write2(brightGreen, ":")
			writeInt(frame.Line)
			writeOffset(brightBlue, offset)
			write("\n")
		}
		isFirst = false
	}
}

func writeOffset(prefix string, offset int) {
	if offset == -1 {
		goto reset
	}
	if !noColor {
		write(prefix)
	}
	write("(")
	writeInt(offset)
	write(")")
reset:
	if !noColor {
		write(reset)
	}
}

func write[Bytes ~[]byte | ~string](b Bytes) {
	p := []byte(b)
	output.Write(*noEscape(&p)) //nolint:errcheck
}

//nolint:all
//go:nosplit
func noEscape[P ~*E, E any](p P) P {
	x := uintptr(unsafe.Pointer(p))
	return P(unsafe.Pointer(x ^ 0))
}

func write2(prefix, s string) {
	if !noColor {
		write(prefix)
	}
	write(s)
}

func write3(prefix, s, suffix string) {
	if !noColor {
		write(prefix)
	}
	write(s)
	if !noColor {
		write(suffix)
	}
}

func writeInt(n int) {
	if n == 0 {
		write("0")
		return
	}
	val := uint(n)
	if n < 0 {
		write("-")
		val = uint(-n)
	}
	var buf [19]byte // len(strconv.FormatInt(math.MaxInt64, 10)) == 19
	i := len(buf) - 1
	for val >= 10 {
		q := val / 10
		buf[i] = byte('0' + val - q*10)
		i--
		val = q
	}
	// val < 10
	buf[i] = byte('0' + val)
	write(buf[i:]) //nolint:errcheck
}
