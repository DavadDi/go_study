package validation

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
)

// Pkg init(), only set log debug
func init() {
	// log.SetFlags(log.Lshortfile | log.LstdFlags)
}

// For use define stuct, without params
// If validate struct has this interface, we will call this interface firstly
type TValidater interface {
	TValidater() error
}

// Interface for Validators
type Validater interface {
	Validater(v interface{}) error
}

// Global var for validation pkg
const (
	ValidTag      = "valid" // Validater tag name
	FuncSeparator = ";"     // Func sparator "required;email"
	ValidIgnor    = "-"     // Igore for validater
	RequiredKey   = "required"
)

var (
	// Init by this pkg. no need rwlock
	ValidatorsMap = map[string]Validater{
		"email":     &EmailChecker{},
		RequiredKey: &RequiredChecker{},
		"url":       &UrlChecker{},
	}

	// Using rwlock avoid race
	CustomValidatorsMap = NewCustomValidators()

	// Output debug info or not
	DebugFlag = false
)

// Pkg debug function, just a wrapper for log
func Debug(arg interface{}) {
	if DebugFlag {
		log.Print(arg)
	}
}

func Debugf(format string, args ...interface{}) {
	if DebugFlag {
		log.Printf(format, args...)
	}
}

// Enable validation debug log
func EnableDebug(flag bool) {
	DebugFlag = flag
}

// Return a new custom validater manager
func NewCustomValidators() *CustomValidators {
	return &CustomValidators{
		ValidatorsMap: make(map[string]Validater),
	}
}

// Function export to add user define Validater
func AddValidater(name string, validater Validater) error {
	if CustomValidatorsMap == nil {
		CustomValidatorsMap = NewCustomValidators()
	}

	return CustomValidatorsMap.AddValidater(name, validater)
}

// Because user can add user define validater, avoid data race, add rwlock
type CustomValidators struct {
	ValidatorsMap map[string]Validater
	sync.RWMutex
}

// If name conflict with CustomValidators, replace.
// conflict with ValidatorsMap, return err
func (cvm *CustomValidators) AddValidater(name string, validater Validater) error {
	// check validater
	if validater == nil {
		return ErrValidater
	}

	// check name conflict
	if ValidatorsMap[name] != nil {
		return ErrValidaterExists
	}

	cvm.Lock()
	cvm.ValidatorsMap[name] = validater
	Debugf("Add custom validater [%s] succeed!", name)
	cvm.Unlock()

	return nil
}

// Return user define validater for name
func (cvm *CustomValidators) FindValidater(name string) (v Validater, ok bool) {
	cvm.RLock()
	v, ok = cvm.ValidatorsMap[name]
	cvm.RUnlock()

	return v, ok
}

// Validation err list
type Validation struct {
	Errors []*Error
}

func NewValidation() *Validation {
	return &Validation{}
}

// Return msg detail Error message
func (mv *Validation) ErrMsg() string {
	buf := bytes.NewBufferString("")

	for _, err := range mv.Errors {
		str := err.String()
		buf.WriteString(str)
	}

	return buf.String()
}

// Clear error, maybe not need
func (mv *Validation) Clear() {
	mv.Errors = nil
}

// Apend error to validtion
func (mv *Validation) AddError(key string, v interface{}, err error) {
	errtmp := &Error{FieldName: key, Value: v, Err: err}
	mv.Errors = append(mv.Errors, errtmp)
}

// Check has errors or not
func (mv *Validation) HasError() bool {
	return len(mv.Errors) != 0
}

// Validiton entry function.
// True: Validiton passed.
// False: Validate don't passed, mv.ErrMsg() contains the detail info.
func (mv *Validation) Validate(obj interface{}) bool {
	if obj == nil {
		return true
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	t := v.Type()

	// Here only accept structs
	if v.Kind() != reflect.Struct {
		err := &ErrOnlyStrcut{Type: v.Type()}
		mv.AddError("Object", obj, err)
		return false
	}

	Debugf("Check struct [%s]", t.Name())

	objvk, ok := obj.(TValidater)
	if ok {
		if err := objvk.TValidater(); err != nil {
			mv.AddError("Object", obj, err)
		}
	}

	for i := 0; i < v.NumField(); i++ {
		tf := t.Field(i) // type field
		vf := v.Field(i) // vaule field

		// Skip Anonymous and private field
		if !tf.Anonymous && len(tf.PkgPath) > 0 {
			continue
		}

		fns := mv.getValidFuns(tf, ValidTag)

		// Already skip ValidIgnor flag, such as "-"
		if len(fns) == 0 {
			continue
		}

		mv.typeCheck(vf, tf, v)
	}

	if mv.HasError() {
		return false
	}

	return true
}

func (mv *Validation) checkRequire(v reflect.Value, t reflect.StructField) error {
	rck := ValidatorsMap[RequiredKey]
	return rck.Validater(v.Interface())
}

// Valid struct field type
func (mv *Validation) typeCheck(v reflect.Value, t reflect.StructField, o reflect.Value) {
	fns := mv.getValidFuns(t, ValidTag)

	// skip
	if len(fns) == 0 {
		return
	}

	// First check all field for required
	if fns[RequiredKey] != nil {
		if err := mv.checkRequire(v, t); err != nil {
			mv.AddError(t.Name, v.Interface(), err)
		}

		delete(fns, RequiredKey)
	}

	switch v.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.String:

		Debugf("\tCheck field [%s]", t.Name)

		for fname := range fns {
			Debugf("CheckerName: [%s]", fname)

			// find custom map and pkg map
			var vck Validater
			var find bool

			if vck, find = CustomValidatorsMap.FindValidater(fname); !find {
				vck = ValidatorsMap[fname]
			}

			if vck == nil {
				err := fmt.Errorf("can't find checker for [%s]", fname)
				mv.AddError(t.Name, v.Interface(), err)
				Debugf("can't find checker for [%s]", fname)
				continue
			}

			err := vck.Validater(v.Interface())
			if err != nil {
				mv.AddError(t.Name, v.Interface(), err)
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
		// only check
		// If the value is a pointer then check its element
		if !v.IsNil() {
			mv.typeCheck(v.Elem(), t, o)
		}

	case reflect.Struct:
		mv.Validate(v.Interface())

	// case reflect.Map: // don't support map now
	default:
		err := fmt.Errorf("UnspportType %s", v.Type())
		mv.AddError(t.Name, v.Interface(), err)
	}

	return
}

// Return fun names and params
func (mv *Validation) getValidFuns(tf reflect.StructField, tag string) map[string]interface{} {
	out := make(map[string]interface{})

	opt, ok := tf.Tag.Lookup(tag)
	if !ok || len(opt) == 0 || opt == ValidIgnor {
		return nil
	}

	for _, value := range strings.Split(opt, FuncSeparator) {
		// omit func has params for now
		value := strings.TrimSpace(value)
		out[value] = struct{}{}
	}

	return out
}

// Not used for now
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
