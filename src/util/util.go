package util

import (
	"fmt"
	"reflect"
	"strings"
)

/*
func ConvertToInterface(t []int) []interface{} {
	s := make([]interface{}, len(t))
	for i, v := range t {
		s[i] = v
	}
}
*/

// type Person struct {
//	 Name string `http:"name"`
//	 Age  int    `http:age`
// }
//
// person := &Person{Name: "Alice", Age: 18 }
//
// v := reflect.ValueOf(person)
// if v.Kind() == reflect.Ptr {
//	 v = v.Elem()
// }
//
// typ := v.Type()
//
// GetTagName(typ.Field(0), "http") == "name"

func GetTagName(field reflect.StructField, tag string) (name string, err error) {
	tagInfo := field.Tag.Get(tag)
	if len(tagInfo) == 0 {
		err = fmt.Errorf("can't get tagInfo for [%s]", tag)
		return
	}

	tagOpts := strings.Split(tagInfo, ",")
	name = tagOpts[0]

	return
}

func GetTagOpt(field reflect.StructField, tag string) (opt string) {
	return field.Tag.Get(tag)
}

// Convert struct to map[string]interface{}, skip kind Ptr nil field
// Only deal with top level field
func ToTagMap(in interface{}, tag string, ignores *Set) (map[string]interface{}, error) {
	if len(tag) == 0 {
		return nil, fmt.Errorf("tag must not be empty")
	}

	out := make(map[string]interface{})

	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// we only accept structs
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("func ToMap only accepts structs; got %T", v)
	}

	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		// gets us a StructField
		tf := typ.Field(i)
		vf := v.Field(i)

		tagName, _ := GetTagName(tf, tag)
		if len(tagName) == 0 {
			continue
		}

		if ignores != nil && ignores.Exist(tagName) {
			continue
		}

		// skip nil ptr field, Only deal the kind of ptr
		if vf.Kind() == reflect.Ptr && vf.IsNil() {
			continue
		}

		out[tagName] = vf.Interface()
	}

	return out, nil
}
