# Go template学习样例

## 1. 简单的模板

	package main

	import (
		"os"
		"text/template"
	)
	
	type Inventory struct {
		Material string
		Count    uint
	}
	
	func main() {
		tpt := `{{.Count}} items are made of {{.Material}}\n`
	
		sweaters := Inventory{"wool", 17}
	
		tmpl, err := template.New("test").Parse(tpt)
		if err != nil {
			panic(err)
		}
	
		if err = tmpl.Execute(os.Stdout, sweaters); err != nil {
			panic(err)
		}
	}


## 2. Array遍历

	package main

	import (
		"html/template"
		"os"
	)
	
	type EntetiesClass struct {
		Name  string
		Value int32
	}
	
	// In the template, we use rangeStruct to turn our struct values
	// into a slice we can iterate over
	const htmlTemplate = `
	{{range $index, $element := .}}
		{{$index}}
			{{range $element}}
				{{.Name}} {{.Value}}
			{{end}}
	{{end}}
	`
	
	func main() {
		data := map[string][]EntetiesClass{
			"Yoga":    {{"Yoga1", 13}, {"Yoga2", 15}},
			"Pilates": {{"Pilates1", 3}, {"Pilates2", 6}, {"Pilates3", 9}},
		}
	
		t := template.New("t")
		t, err := t.Parse(htmlTemplate)
		if err != nil {
			panic(err)
		}
	
		err = t.Execute(os.Stdout, data)
		if err != nil {
			panic(err)
		}
	}



## 3. Map遍历

### 3.1 Range 遍历

按照map的rang来进行访问

	package main

	import (
	    "fmt"
	    "os"
	    "text/template"
	
	)
	
	func main() {
	    fmt.Println("Hello, playground")
	
	    const templ = `Here is what they said
	    {{ range $key, $value := . }}
	        {{ $key }}:{{ $value }}
	    {{end}}
	    `
	    x := map[string]string{
	        "Danny": "The guy really talked about my first time out with you",
	        "Doug":  "Well he said I'm really amazing, I did not believe at first",
	
	    }
	
	    t, err := template.New("index.html").Parse(templ)
	    if err != nil {
	        fmt.Println("Could not parse template:", err)
	        return
	
	    }
	    t.Execute(os.Stdout, x)
	
	}

### 3.2 Key 遍历

按照map的key来进行访问

	package main

	import (
		"os"
		"text/template"
	)
	
	// show how to get map key
	func main() {
	
		// map value:	{{index .mymap "key"}}
		tpt := `
			{{if eq "hello" (index .mymap "key")}} hello {{else}} not hello {{end}} 
		`
	
		t := template.Must(template.New("").Parse(tpt))
	
		value := map[string]interface{}{"mymap": map[string]string{"key": "value"}}
	
		t.Execute(os.Stdout, value)
	}


参考：

1. 	[Access a map value using a variable key in a Go template](http://stackoverflow.com/questions/26152088/access-a-map-value-using-a-variable-key-in-a-go-template)

2. [How to get map value by key without range action (htm/text templates)? Golang](http://stackoverflow.com/questions/28010961/how-to-get-map-value-by-key-without-range-action-htm-text-templates-golang)