package helpers

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	respConst "codebase/internal/constant/response"
	"codebase/models"
)

type (
	LogEntry struct {
		ErrConst  string `json:"error_const"`
		ErrInfo   string `json:"error"`
		ErrWrap   bool   `json:"error_wrap"`
		Component string `json:"component"`
		SrcFile   string `json:"src_file"`
		SrcFunc   string `json:"src_func"`
	}

	ErrorImpl struct {
		sharedHolder *models.SharedHolder
	}

	Error interface {
		Wrap(errConst, errInfo string) error
	}
)

func (l *LogEntry) Error() string {
	return l.ErrConst + ": " + l.ErrInfo
}

func NewError(sharedHolder *models.SharedHolder) Error {
	return &ErrorImpl{
		sharedHolder: sharedHolder,
	}
}

func (e *ErrorImpl) Wrap(errConst, errInfo string) error {
	var (
		component, srcFile, srcFunc string
		errWrap                     bool
		allowlistDomains            = []string{
			respConst.VALIDATION_ERROR,
		}
	)

	if pc, file, line, ok := runtime.Caller(1); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			srcFunc = fn.Name()
		}

		funcParts := strings.Split(srcFunc, ".")
		srcFunc = funcParts[len(funcParts)-1]

		fileParts := strings.Split(file, "/")
		file = fileParts[len(fileParts)-1]
		component = fileParts[len(fileParts)-2]
		srcFile = fmt.Sprintf("%s:%d", file, line)
	}

	if SlicesContains(allowlistDomains, errConst) {
		errWrap = true
	}

	return &LogEntry{
		ErrConst:  errConst,
		ErrInfo:   errInfo,
		ErrWrap:   errWrap,
		Component: component,
		SrcFile:   srcFile,
		SrcFunc:   srcFunc,
	}
}

// Coded returns err unchanged when it already carries a response code (e.g. it came back out
// of a transaction), and wraps anything else as GENERAL_ERROR.
func Coded(errs Error, err error) error {
	var e *LogEntry
	if errors.As(err, &e) {
		return e
	}
	return errs.Wrap(respConst.GENERAL_ERROR, err.Error())
}
