package main

import (
    "fmt"
    "log"

    "github.com/boltdb/bolt"
)

var world = []byte("world")

func main() {
    db, err := bolt.Open("./bolt.db", 0644, nil)
    if err != nil {
        log.Fatal(err)
    }

    defer db.Close()

    key := []byte("hello")
    value := []byte("Hello World!")

    // store some data
    err = db.Update(func(tx *bolt.Tx) error {
        bucket, err := tx.CreateBucketIfNotExists(world)
        if err != nil {
            return err
        }

        err = bucket.Put(key, value)
        if err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        log.Fatal(err)
    }

    // retrieve the data
    err = db.View(func(tx *bolt.Tx) error {
        bucket := tx.Bucket(world)
        if bucket == nil {
            return fmt.Errorf("Bucket %q not found!", world)
        }

        val := bucket.Get(key)
        fmt.Println(string(val))

        return nil
    })

    if err != nil {
        log.Fatal(err)
    }
}

// output:
// Hello World!
