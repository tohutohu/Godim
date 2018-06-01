package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

const (
	ioctlGetTermios = unix.TIOCGETA
	ioctlSetTermios = unix.TIOCGETA
)

func main() {
	terminal.MakeRaw(int(os.Stdin.Fd()))
	enableRawModw(os.Stdin.Fd())
	fmt.Println("po")

	bufCh := make(chan []byte, 1024)
	go readBuffer(bufCh)
	for {
		ch := <-bufCh
		for _, c := range ch {
			fmt.Println(string(c))
		}
	}
}

func enableRawModw(fd uintptr) error {
	termios, err := unix.IoctlGetTermios(int(fd), ioctlGetTermios)
	if err != nil {
		panic(err)
	}
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.ISTRIP | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN | unix.ISIG | unix.ECHONL
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8

	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	err = unix.IoctlSetTermios(int(fd), ioctlSetTermios, termios)
	if err != nil {
		panic(err)
	}
	return nil
}

func readBuffer(ch chan []byte) {
	buf := make([]byte, 100)
	for {
		if n, err := unix.Read(int(os.Stdin.Fd()), buf); err == nil {
			ch <- buf[:n]
		}
	}
}
