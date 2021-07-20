package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

func main() {
	fmt.Println(runtime.NumCPU())
	wg := sync.WaitGroup{}

	setupDumpStackTrap()

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(num int) {
			time.Sleep(120 * time.Second)
			fmt.Println("Hello world [%d]", num)
		}(i)
	}

	wg.Wait()
}

func setupDumpStackTrap() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, unix.SIGUSR1)
	go func() {
		for range c {
			dump(os.Stderr)
		}

	}()
}

func dump(f *os.File) error {
	var (
		buf       []byte
		stackSize int
	)
	bufferLen := 16384
	for stackSize == len(buf) {
		buf = make([]byte, bufferLen)
		stackSize = runtime.Stack(buf, true)
		bufferLen *= 2
	}
	buf = buf[:stackSize]
	if _, err := f.Write(buf); err != nil {
		return errors.Errorf("failed to write goroutine stacks")
	}
	return nil
}
