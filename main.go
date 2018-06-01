package main

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

const (
	ioctlGetTermios    = unix.TIOCGETA
	ioctlSetTermios    = unix.TIOCSETA
	ioctlGetWindowSize = unix.TIOCGWINSZ
)

func main() {
	winSize, err := getWindowSize(os.Stdin.Fd())
	if err != nil {
		panic(err)
	}
	fmt.Println(winSize)
	oldTerm, err := enableRawMode(os.Stdin.Fd())
	if err != nil {
		panic(err)
	}

	bufCh := make(chan []byte, 1024)
	go readBuffer(bufCh)
LOOP:
	for {
		ch := <-bufCh
		for _, c := range ch {
			if err := inputHandle(c); err != nil {
				break LOOP
			}
		}
	}
	unix.IoctlSetTermios(int(os.Stdin.Fd()), ioctlSetTermios, oldTerm)
}

func inputHandle(c byte) error {
	if c == 4 || c == 3 {
		return errors.New("Exit")
	}
	fmt.Println(string(c))
	return nil
}

func getWindowSize(fd uintptr) (*unix.Winsize, error) {
	winSize, err := unix.IoctlGetWinsize(int(fd), ioctlGetWindowSize)
	if err != nil {
		return nil, err
	}
	return winSize, nil
}

func enableRawMode(fd uintptr) (*unix.Termios, error) {
	termios, err := unix.IoctlGetTermios(int(fd), ioctlGetTermios)
	if err != nil {
		return nil, err
	}
	oldTerm := *termios

	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(int(fd), ioctlSetTermios, termios); err != nil {
		return nil, err
	}
	return &oldTerm, nil
}

func readBuffer(ch chan []byte) {
	buf := make([]byte, 100)
	for {
		if n, err := unix.Read(int(os.Stdin.Fd()), buf); err == nil {
			ch <- buf[:n]
		}
	}
}
