// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package stacktrace

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"sync"
	"unsafe"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

var (
	noColor bool
	output  io.Writer
	enabled = true
	mu      sync.Mutex
)

func init() {
	fd := os.Stderr.Fd()
	noColor = os.Getenv("NO_COLOR") != "" || (!isatty.IsTerminal(fd) && !isatty.IsCygwinTerminal(fd))
	if runtime.GOOS != "windows" || noColor {
		output = os.Stderr
	} else {
		output = colorable.NewColorableStderr()
	}
}

func Enable() {
	mu.Lock()
	defer mu.Unlock()
	enabled = true
}

func Disable() {
	mu.Lock()
	defer mu.Unlock()
	enabled = false
}

func Go(f func(), print func(w io.Writer, r any), predicate func(frame runtime.Frame) bool) {
	go func() {
		defer Handle(true, print, predicate)
		f()
	}()
}

func Handle(exit bool, print func(w io.Writer, r any), predicate func(frame runtime.Frame) bool) {
	r := recover()
	if r == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if !enabled {
		return
	}
	if exit {
		defer os.Exit(2)
	}

	write3(bold+brightCyan, "panic: ", reset)
	writeColor(bold + brightBlue)
	if print != nil {
		print(output, r)
	} else {
		fmt.Fprint(output, r) //nolint:errcheck
	}
	write2(reset, "\n")

	isFirst := true
	for frame := range CallStack(2, predicate) {
		packagePath, functionName := SplitFunctionPath(frame.Function)
		dir, name := path.Split(frame.File)
		offset := -1
		if frame.Func != nil {
			funcFile, funcLine := frame.Func.FileLine(frame.Func.Entry())
			if funcFile == frame.File && frame.Line >= funcLine {
				offset = frame.Line - funcLine
			}
		}

		if isFirst {
			write3(red, "->", reset)
			write("  at ")
			write2(bold+brightYellow, packagePath)
			write2(bold+brightGreen, functionName)
			writeOffset(bold+brightBlue, offset)
			write("\n")
			write2(red, "->")
			write("       ")
			write2(bold+brightWhite, dir)
			write2(bold+brightCyan, name)
			write2(bold+brightGreen, ":")
			writeInt(frame.Line)
			write2(reset, "\n\n")
		} else {
			write("    at ")
			write2(yellow, packagePath)
			write2(brightGreen, functionName)
			writeOffset(brightBlue, offset)
			write("\n         ")
			write2(brightWhite, dir)
			write2(brightCyan, name)
			write2(brightGreen, ":")
			writeInt(frame.Line)
			write2(reset, "\n")
		}
		isFirst = false
	}
}

func writeOffset(prefix string, offset int) {
	if offset < 0 {
		goto reset
	}
	write2(prefix, "+")
	writeInt(offset)
reset:
	writeColor(reset)
}

func write[Bytes ~[]byte | ~string](b Bytes) {
	p := []byte(b)
	output.Write(*noEscape(&p)) //nolint:errcheck
}

//go:nosplit
func noEscape[P ~*E, E any](p P) P {
	x := uintptr(unsafe.Pointer(p))
	return P(unsafe.Pointer(x ^ 0)) //nolint:all
}

func writeColor(s string) {
	if !noColor {
		write(s)
	}
}

func write2(prefix, s string) {
	writeColor(prefix)
	write(s)
}

func write3(prefix, s, suffix string) {
	writeColor(prefix)
	write(s)
	writeColor(suffix)
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
	buf[i] = byte('0' + val) // val < 10
	write(buf[i:])
}
