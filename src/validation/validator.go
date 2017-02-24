package validation

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strings"
)

// For use define stuct, maybe provide another interface without params
type TValidater interface {
	TValidater() error
}

type Validater interface {
	Validater(v interface{}) error
}

const (
	ValidTag     = "valid"
	Funseparator = ","
	ValidIgnor   = "-"
)

// init by this pkg. no need lock
var ValidatorsMap = map[string]Validater{
	"email":    &EmailChecker{},
	"required": &RequiredChecker{},
	"url":      &UrlChecker{},
}

// TODO
// need rwlock
// var CunstomValidatorsMap

type Validation struct {
	Errors []*Error
}

func NewValidation() *Validation {
	return &Validation{}
}

// return error msg
func (mv *Validation) ErrMsg() string {
	buf := bytes.NewBufferString("")

	for _, err := range mv.Errors {
		str := err.String()
		buf.WriteString(str)
	}

	return buf.String()
}

// clear error, maybe not need
func (mv *Validation) Clear() {
	mv.Errors = nil
}

// apend error to validtion
func (mv *Validation) AddError(err *Error) {
	mv.Errors = append(mv.Errors, err)
}

// check has errors or not
func (mv *Validation) HasError() bool {
	return len(mv.Errors) != 0
}

// validiton entry function
// true: check passed
// false: don't passed, err contains the detail info
func (mv *Validation) Validate(obj interface{}) (bool, error) {
	if obj == nil {
		return true, nil
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	t := v.Type()

	// here only accept structs
	if v.Kind() != reflect.Struct {
		return false, &ErrOnlyStrcut{Type: v.Type()}
	}

	objvk, ok := obj.(TValidater)
	if ok {
		if err := objvk.TValidater(); err != nil {
			mv.AddError(&Error{FieldName: "Object", Value: obj, Err: err})
		}
	}

	for i := 0; i < v.NumField(); i++ {
		tf := t.Field(i) // type field
		vf := v.Field(i) // vaule field

		// skip Anonymous and private field
		if !tf.Anonymous && len(tf.PkgPath) > 0 {
			continue
		}

		fns := mv.getValidFuns(tf, ValidTag)

		// already skip ValidIgnor flag, such as "-"
		if len(fns) == 0 {
			continue
		}

		mv.typeCheck(vf, tf, v)
	}

	if mv.HasError() {
		return false, nil
	}

	return true, nil
}

// valid struct field type
func (mv *Validation) typeCheck(v reflect.Value, t reflect.StructField, o reflect.Value) {
	fns := mv.getValidFuns(t, ValidTag)

	// skip
	if len(fns) == 0 {
		return
	}

	switch v.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.String:

		log.Printf("typeCheck: %s", v.Kind())
		for fname := range fns {
			log.Printf("CheckName: [%s]", fname)
			if vck, ok := ValidatorsMap[fname]; ok {
				err := vck.Validater(v.Interface())
				if err != nil {
					mv.AddError(&Error{FieldName: t.Name, Value: v.Interface(), Err: err})
				}
			} else {
				log.Printf("can't find checker for [%s]", fname)
			}
		}

	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if v.Index(i).Kind() != reflect.Struct {
				mv.typeCheck(v.Index(i), t, o)
			} else {
				mv.Validate(v.Index(i).Interface())
			}
		}

	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if v.Index(i).Kind() != reflect.Struct {
				mv.typeCheck(v.Index(i), t, o)
			} else {
				mv.Validate(v.Index(i).Interface())

			}
		}

	case reflect.Interface:
		// If the value is an interface then encode its element
		if !v.IsNil() {
			mv.Validate(v.Interface())
		}

	case reflect.Ptr:
		// If the value is a pointer then check its element
		if !v.IsNil() {
			mv.typeCheck(v.Elem(), t, o)
		}

	case reflect.Struct:
		mv.Validate(v.Interface())

	// case reflect.Map: // don't support map now
	default:
		err := &Error{FieldName: t.Name, Value: v.Interface(), Err: fmt.Errorf("UnspportType %s", v.Type())}
		mv.AddError(err)
	}

	return
}

// return fun names and params
func (mv *Validation) getValidFuns(tf reflect.StructField, tag string) map[string]interface{} {
	out := make(map[string]interface{})

	opt, ok := tf.Tag.Lookup(tag)
	if !ok || len(opt) == 0 || opt == ValidIgnor {
		return nil
	}

	for _, value := range strings.Split(opt, Funseparator) {
		// omit func has params
		out[value] = struct{}{}
	}

	return out
}

// not used for now
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String, reflect.Array:
		return v.Len() == 0
	case reflect.Map, reflect.Slice:
		return v.Len() == 0 || v.IsNil()
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}

	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}
