package util

import (
	"reflect"
	"testing"
)

var person = struct {
	Name  string `http:"name"`
	Age   int    `http:"age"`
	Empty string
}{
	Name:  "Alice",
	Age:   18,
	Empty: "",
}

func TestGetTagName(t *testing.T) {
	/*
		tests := struct {
			Name string,
			ExpectNum int,
			ExpectStr string,
		}{
			{"name", 1, "name"},
			{"age", 1, "age"},
		}
	*/
	v := reflect.ValueOf(person)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	typ := v.Type()

	field, ok := typ.FieldByName("Name")
	if !ok {
		t.Fatal("Get Person field by name failed")
	}

	// Test exist tag
	name, err := GetTagName(field, "http")
	if err != nil {
		t.Errorf("Get tag[http] failed. [%s]", err)
	}

	if name != "name" {
		t.Errorf("GetTagName should got 'name', but got %#v", name)
	}

	// Test don't exist tag
	field, ok = typ.FieldByName("Empty")
	if !ok {
		t.Fatal("Get Person field by name failed")
	}

	name, err = GetTagName(field, "http")
	if err == nil {
		t.Errorf("Get tag[http] should failed. [%s]", name)
	}

	if len(name) != 0 {
		t.Errorf("GetTagName should got '', but got %#v", name)
	}
}

func TestToTagMap(t *testing.T) {
	out, err := ToTagMap(&person, "http", nil)
	if err != nil {
		t.Fatal("Call ToTagMap failed")
	}

	size := len(out)
	if size != 2 {
		t.Errorf("Should got num 2, but got [%d], %v", size, out)
	}

	name, ok := out["name"]
	if !ok || name.(string) != person.Name {
		t.Errorf("Should got name field is [%s], but got [%s]", person.Name, name.(string))
	}

	age, ok := out["age"]
	if !ok || age.(int) != person.Age {
		t.Errorf("Should got name field is [%d], but got [%d]", person.Age, age.(int))
	}
}

func TestToTagMapWithIgore(t *testing.T) {
	out, err := ToTagMap(&person, "http", NewSet("name"))
	if err != nil {
		t.Fatal("Call ToTagMap failed")
	}

	size := len(out)
	if size != 1 {
		t.Errorf("Should got num 1, but got [%d], %v", size, out)
	}

	age, ok := out["age"]
	if !ok || age.(int) != person.Age {
		t.Errorf("Should got name field is [%d], but got [%d]", person.Age, age.(int))
	}

}

func TestToTagMapTagNil(t *testing.T) {
	_, err := ToTagMap(&person, "", nil)
	if err == nil {
		t.Fatal("Call ToTagMap should failed")
	}
}
