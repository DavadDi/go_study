package validation

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrBadUrlFormat   = errors.New("url format is not valid")
	ErrBadEmailFormat = errors.New("email addr is not valid")
	ErrRequired       = errors.New("can't be empty or zero")
)

type Error struct {
	FieldName string
	Value     interface{}
	Err       error
}

func (err *Error) String() string {
	return fmt.Sprintf("[%s] check failed [%s] [%#v]", err.FieldName, err.Err.Error(), err.Value)
}

type ErrUnsupportedType struct {
	Type reflect.Type
}

func (err *ErrUnsupportedType) Error() string {
	return "validition unsupported type: " + err.Type.String()

}

type ErrOnlyStrcut struct {
	Type reflect.Type
}

func (err *ErrOnlyStrcut) Error() string {
	return "validition only support struct, but got type: " + err.Type.String()

}
