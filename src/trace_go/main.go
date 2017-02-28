package main

import (
	"fmt"
)

func Trace(s string) string {
	fmt.Println("Entering: ", s)
	return s
}

func Untrace(s string) {
	fmt.Println("Leaving:", s)
}

// used anonymous func
func TraceFunc(s string) func() {
	fmt.Println("Entering: " + s)
	return func() {
		fmt.Println("Leaving: " + s)
	}
}

func main() {
    // annonymouse usage
	// defer TraceFunc("main")()

    // netsted usage
	defer Untrace(Trace("main"))
	fmt.Println("Hello, playground")

}
