package util

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

func WrapError(in interface{}, a ...any) error {
	_, full, line, _ := runtime.Caller(1)
	root, err := os.Getwd()
	if err != nil {
		return errors.New("sys error:" + err.Error())
	}
	file := strings.Replace(full, root+"/", "", 1)
	prefix := fmt.Sprintf("%s:%d →\n", file, line)
	switch v := in.(type) {
	case error:
		return errors.New(prefix + v.Error())
	case string:
		return errors.New(fmt.Sprintf(prefix+v, a...))
	case nil:
		return nil
	default:
		return errors.New(prefix + " unknown in type")
	}
}
