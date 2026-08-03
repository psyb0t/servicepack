package ctxerrors

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime"
)

// CTXError holds the wrapped error and additional context.
type CTXError struct {
	err      error
	message  string
	file     string
	line     int
	funcName string
}

// New creates a new error with context but without wrapping another error.
func New(message string) error {
	// Skip New() to get user's caller
	framesToSkip := 1

	file, line, funcName := getCallerInfo(framesToSkip)

	return &CTXError{
		message:  message,
		file:     file,
		line:     line,
		funcName: funcName,
	}
}

// Wrap wraps an error with context information (file, line, and function name).
func Wrap(err error, message string) error {
	// Skip Wrap() and wrap() to get user's caller
	framesToSkip := 2

	return wrap(err, message, framesToSkip)
}

// Wrapf wraps an error with context information (file, line, and function name).
func Wrapf(err error, format string, args ...any) error {
	// Skip Wrapf() and wrap() to get user's caller
	framesToSkip := 2

	return wrap(err, fmt.Sprintf(format, args...), framesToSkip)
}

// Join combines several errors into one that carries the call site, for the
// case where an operation fans out and more than one branch can fail
// independently — several sinks written to, several items in a batch. nil
// errors are ignored, and Join returns nil when every error is nil.
//
// The result unwraps to what the standard library's errors.Join produces, so
// errors.Is and errors.As still find any of the joined errors.
func Join(errs ...error) error {
	joined := errors.Join(errs...)
	if joined == nil {
		return nil
	}

	count := 0

	for _, err := range errs {
		if err != nil {
			count++
		}
	}

	// Skip Join() to get user's caller
	framesToSkip := 1

	file, line, funcName := getCallerInfo(framesToSkip)

	return &CTXError{
		err:      joined,
		message:  fmt.Sprintf("%d errors", count),
		file:     file,
		line:     line,
		funcName: funcName,
	}
}

func wrap(err error, message string, skip int) error {
	if err == nil {
		// log three frames so callers can see the actual call chain that
		// produced the nil, not just the immediate caller
		debugFrame1 := 2
		debugFrame2 := 3
		debugFrame3 := 4

		pc, file, line, _ := runtime.Caller(debugFrame1)
		funcName := runtime.FuncForPC(pc).Name()

		pc2, file2, line2, _ := runtime.Caller(debugFrame2)
		funcName2 := runtime.FuncForPC(pc2).Name()

		pc3, file3, line3, _ := runtime.Caller(debugFrame3)
		funcName3 := runtime.FuncForPC(pc3).Name()

		slog.Error("Trying to wrap a nil error",
			"sourceFile1", fmt.Sprintf("%s:%d", file, line),
			"sourceFunc1", funcName,
			"sourceFile2", fmt.Sprintf("%s:%d", file2, line2),
			"sourceFunc2", funcName2,
			"sourceFile3", fmt.Sprintf("%s:%d", file3, line3),
			"sourceFunc3", funcName3,
		)

		return nil
	}

	err = translate(err)

	file, line, funcName := getCallerInfo(skip)

	return &CTXError{
		err:      err,
		message:  message,
		file:     file,
		line:     line,
		funcName: funcName,
	}
}

// Unwrap retrieves the underlying error, if any.
func (e *CTXError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.err
}

// Error returns the formatted error message, including file and function details.
func (e *CTXError) Error() string {
	if e == nil {
		return ""
	}

	if e.err != nil {
		return fmt.Sprintf(
			"%s: %s [%s:%d in %s]",
			e.message, e.err, e.file, e.line, e.funcName,
		)
	}

	return fmt.Sprintf(
		"%s [%s:%d in %s]",
		e.message, e.file, e.line, e.funcName,
	)
}

// getCallerInfo retrieves file, line, and function name where the error was created.
func getCallerInfo(skip int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "", 0, ""
	}

	funcName := runtime.FuncForPC(pc).Name()

	return file, line, funcName
}
