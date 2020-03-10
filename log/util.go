package log

import (
	"runtime"
	"strconv"
	"strings"
)

// LazyEval can be used to evaluate an argument under a correct log level.
type LazyEval func() string

func (l LazyEval) String() string {
	return l()
}

// DoLazyEval returns LazyEval. Unnecessary evalution can be prevented by using
// "%v" format string,
func DoLazyEval(c func() string) LazyEval {
	return LazyEval(c)
}

// SkipCaller returns caller's location (file and line) to help debug.
// This passes a skip number, which is given by an arg, of callers
func SkipCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "?"
	}
	return file + ":" + strconv.Itoa(line)
}

// PanicInvoker returns caller's location (file and line) that causes panic.
func PanicInvoker(skipTip int) string {
	for {
		_, file, line, ok := runtime.Caller(skipTip)
		if !ok {
			return "?" // unknown or an end of stack
		} else if strings.HasSuffix(file, "runtime/panic.go") {
			skipTip++
			continue //skip system callstack
		}

		return file + ":" + strconv.Itoa(line)
	}
}
