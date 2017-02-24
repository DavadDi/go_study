package validation

import (
	"reflect"
	"testing"
)

type Person struct {
	Name     string    `valid:"required"`
	Email    string    `valid:"required,email"`
	Age      int       `valid:"-"`
	Sex      int       ``
	WebSites []*string `valid:"url"`
}

func (p *Person) TValidater() error {
	return nil
}

var (
	web1   = "http://www.do1618.com"
	web2   = "www"
	person = &Person{Name: "dave", Email: "dwh0403@163.com", WebSites: []*string{&web1, &web2}}
)

func TestGetFuns(t *testing.T) {
	v := reflect.ValueOf(*person)

	tests := []struct {
		Name       string
		ExpectNum  int
		ExpectStrs []string
	}{
		{"Name", 1, []string{"required"}},
		{"Email", 2, []string{"required", "email"}},
		{"Age", 0, nil}, // - skip count
		{"Sex", 0, nil},
	}

	typ := v.Type()
	valider := NewValidation()

	for _, test := range tests {
		field, ok := typ.FieldByName(test.Name)
		if !ok {
			t.Fatalf("Get field by [%s] failed", test.Name)
		}

		// Test exist tag
		// funct list
		fns := valider.getValidFuns(field, ValidTag)

		if len(fns) != test.ExpectNum {
			t.Errorf("Get tag vliad failed. should get [%d], got [%d] %v",
				test.ExpectNum, len(fns), fns)
		}

		// Test fun name list
		for _, name := range test.ExpectStrs {
			if _, ok := fns[name]; !ok {
				t.Errorf("Func name [%s] should in list. but got %v", name, fns)
			}
		}
	}
}

func TestValidation(t *testing.T) {
	valider := NewValidation()
	res, _ := valider.Validate(person)

	if !res {
		t.Errorf("Validate person failed:\n %s", valider.ErrMsg())
	}
}

// Test struct interface
func TestValidationIf(t *testing.T) {
	validor := NewValidation()
	if res, _ := validor.Validate(person); !res {
		t.Errorf("Validate Interface failed:\n %s", validor.ErrMsg())
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		Email  string
		Expect bool
	}{
		{"dwh0403@163.com", true},
		{"aaa@", false},
		{"dwh0403", false},
		{"13455", false},
	}

	eck := &EmailChecker{}

	for _, test := range tests {
		err := eck.Validater(test.Email)
		res := (err == nil)

		if res != test.Expect {
			t.Errorf("Email check failed. [%s] should [%t]",
				test.Email, test.Expect)
		}
	}
}
