package validation

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	// Error for Validater impl
	ErrBadUrlFormat   = errors.New("url format is not valid")
	ErrBadEmailFormat = errors.New("email format is not valid")
	ErrRequired       = errors.New("field can't be empty or zero")

	// Error for Validater
	ErrValidater        = errors.New("validater should not be nil")
	ErrValidaterNoFound = errors.New("validater not found")
	ErrValidaterExists  = errors.New("validater exist")
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

func NewErrWrongType(expect string, value interface{}) error {
	return &ErrWrongExpectType{
		ExpectType: expect,
		PassValue:  value,
	}
}

type ErrWrongExpectType struct {
	ExpectType string
	PassValue  interface{}
}

func (err *ErrWrongExpectType) Error() string {
	return fmt.Sprintf("expect type %s, but got %T", err.ExpectType, err.PassValue)
}
