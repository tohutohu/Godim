package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

func main() {
	terminal.MakeRaw(int(os.Stdin.Fd()))
	ch := make(chan []byte, 100)
	go func() {
		buf := make([]byte, 100)
		for {
			n, err := unix.Read(int(os.Stdin.Fd()), buf)
			if err == nil {
				ch <- buf[:n]
			}
		}
	}()

Loop:
	for {
		select {
		case cc := <-ch:
			fmt.Println(cc)
			if cc[0] == 4 {
				break Loop
			}
		}
	}
}
