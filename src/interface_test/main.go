package main

import (
    "fmt"
    "strconv"

)

type Stringer interface {
    String() string

}

func ToString(any interface{}) string {
    if v, ok := any.(Stringer); ok {
        return v.String()

    }
    switch v := any.(type) {
    case int:
        return strconv.Itoa(v)
    case float32:
        return strconv.FormatFloat(float64(v), 'g', -1, 32)

    }
    return "???"

}

type Binary uint64
type Binary32 uint32

func (i Binary) String() string {
    return strconv.FormatUint(i.Get(), 2)

}

func (i Binary) Get() uint64 {
    return uint64(i)

}

func (i Binary32) String() string {
    return strconv.FormatUint(uint64(i.Get()), 2)

}

func (i Binary32) Get() uint64 {
    return uint64(i)

}

func main() {

    b := Binary(200)
    b32 := Binary32(200)

    s := Stringer(b)
    s32 := Stringer(b32)

    any := (interface{})(b)

    fmt.Println(s.String())
    fmt.Println(s32.String())

    fmt.Println(ToString(any))

    _ = "breakpoint"

    var nothing interface{}

    nothing = uint64(200)
    fmt.Println(nothing)

    var face Stringer

    fmt.Println(face)
}
