package bench

import (
    "bytes"
    "encoding/gob"
    "encoding/json"
    "net"
    "testing"
    "time"

)

type Foo struct {
    Name      string
    LastLogin time.Time
    Address   net.IP
    Ids       []int

}

var (
    buf = &bytes.Buffer{}

    u = Foo{
        `Lorem ipsum dolor sitamet, consectetur adipisicing elit, sed do eiusmod
        tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
        quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
        consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum
        dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident,
        sunt in culpa qui officia deserunt mollit anim id est laborum.`,
        time.Now(),
        net.IPv4(127, 0, 0, 1),
        []int{1, 2, 3, 4, 5, 6, 7, 8, 9},

    }

)

func BenchmarkGobEncode(b *testing.B) {
    enc := gob.NewEncoder(buf)

    benchEncode(enc, b)

}

func BenchmarkJsonEncode(b *testing.B) {
    enc := json.NewEncoder(buf)

    benchEncode(enc, b)

}

func benchEncode(enc Encoder, b *testing.B) {
    // get the initial compile out of the way
    if err := enc.Encode(u); err != nil {
        b.Fatal(err)

    }
    buf.Reset()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        if err := enc.Encode(u); err != nil {
            b.Fatal(err)

        }

        b.StopTimer()
        buf.Reset()
        b.StartTimer()

    }

}

func BenchmarkGobDecode(b *testing.B) {
    buf.Reset()
    enc := gob.NewEncoder(buf)
    dec := gob.NewDecoder(buf)

    benchDecode(dec, enc, b)

}

func BenchmarkJsonDecode(b *testing.B) {
    buf.Reset()
    enc := json.NewEncoder(buf)
    dec := json.NewDecoder(buf)

    benchDecode(dec, enc, b)

}

func benchDecode(dec Decoder, enc Encoder, b *testing.B) {
    if err := enc.Encode(u); err != nil {
        b.Fatal(err)

    }

    var Foo Foo

    // get the initial compile out of the way
    if err := dec.Decode(&Foo); err != nil {
        b.Fatal(err)

    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        b.StopTimer()
        buf.Reset()
        if err := enc.Encode(u); err != nil {
            b.Fatal(err)

        }
        b.StartTimer()

        if err := dec.Decode(&Foo); err != nil {
            b.Fatal(err)

        }

    }


}

type Decoder interface {
    Decode(v interface{}) error

}

type Encoder interface {
    Encode(v interface{}) error

}

