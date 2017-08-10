package main

import "fmt"
import "my_version/version"

var Version = "not provider"

func  main() {
    fmt.Println(Version)
    fmt.Println(version.GitSHA)
}
