package main

//void SayHello(const char *s);
import "C"

func main() {
	C.SayHello(C.CString("Hello\n"))
}

/*
// export SayHello
func SayHello(s string) {
	fmt.Print(s)
}
*/
