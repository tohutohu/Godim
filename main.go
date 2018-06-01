package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	ioctlGetTermios    = unix.TIOCGETA
	ioctlSetTermios    = unix.TIOCSETA
	ioctlGetWindowSize = unix.TIOCGWINSZ
)

type Panel struct {
	UnSave    bool
	FileName  string
	File      *os.File
	Rows      []string
	CursorPos CursorPosition
}

type CursorPosition struct {
	X int
	Y int
}

type Config struct {
	Number bool
}

type E struct {
	WinSize    *unix.Winsize
	OldTermios *unix.Termios
	Panels     []*Panel
	CurPanel   *Panel
	Config     *Config
}

var (
	env *E
)

func main() {
	_env, err := initializeEditor()
	if err != nil {
		panic(err)
	}
	env = _env

	bufCh := make(chan []byte, 1024)
	go readBuffer(bufCh)
	env.refleshScreen()
LOOP:
	for {
		ch := <-bufCh
		if err := env.inputHandle(ch); err != nil {
			break LOOP
		}
		env.refleshScreen()
	}
	beforeExit()
}

func initializeEditor() (*E, error) {
	env := &E{
		Config: &Config{
			Number: false,
		},
	}
	if len(os.Args) < 2 {
		return nil, errors.New("Usage: godim <filename>")
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	text, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	env.Panels = append(env.Panels, &Panel{
		UnSave:   false,
		FileName: os.Args[1],
		File:     file,
		Rows:     strings.Split(string(text), "\n"),
	})

	env.CurPanel = env.Panels[0]

	env.WinSize, err = getWindowSize(os.Stdin.Fd())
	if err != nil {
		panic(err)
	}
	env.OldTermios, err = enableRawMode(os.Stdin.Fd())
	if err != nil {
		panic(err)
	}

	return env, nil
}

func beforeExit() {
	stat, err := env.CurPanel.File.Stat()
	if err != nil {
		fmt.Println(err)
	}

	err = ioutil.WriteFile(env.CurPanel.FileName, []byte(strings.Join(env.CurPanel.Rows, "\n")), stat.Mode())
	if err != nil {
		fmt.Println(err)
	}
	unix.IoctlSetTermios(int(os.Stdin.Fd()), ioctlSetTermios, env.OldTermios)
}

func (env *E) refleshScreen() {
	outText := "\x1b[?25l"
	outText += "\x1b[H"
	for _, row := range env.CurPanel.Rows {
		outText += row
		outText += strings.Repeat(" ", int(env.WinSize.Col)-len(row))
	}

	rowPadding := 0
	if int(env.WinSize.Row-2) >= len(env.CurPanel.Rows) {
		rowPadding = int(env.WinSize.Row-2) - len(env.CurPanel.Rows)
	}

	for i := 0; i < rowPadding; i++ {
		outText += strings.Repeat(" ", int(env.WinSize.Col))
	}
	outText += fmt.Sprintf("%d %d", env.CurPanel.CursorPos.X, env.CurPanel.CursorPos.Y) + strings.Repeat(" ", int(env.WinSize.Col-10))
	outText += strings.Repeat(" ", int(env.WinSize.Col))
	outText += fmt.Sprintf("\x1b[%d;%dH", env.CurPanel.CursorPos.Y+1, env.CurPanel.CursorPos.X+1)
	outText += "\x1b[?25h"
	fmt.Print(outText)
}

func (env *E) inputHandle(in []byte) error {
	if in[0] == 27 && in[1] == 91 {
		switch in[2] {
		case 'A':
			env.CurPanel.CursorPos.Y--
		case 'B':
			env.CurPanel.CursorPos.Y++
		case 'C':
			env.CurPanel.CursorPos.X++
		case 'D':
			env.CurPanel.CursorPos.X--
		}
	} else {
		for _, c := range in {
			switch c {
			case 4:
				return errors.New("Exit")

			case 3:
				return errors.New("Exit")

			case 127:
				if env.CurPanel.CursorPos.X == 0 {
					if env.CurPanel.CursorPos.Y == 0 {
						break
					}
					row := env.CurPanel.Rows[env.CurPanel.CursorPos.Y]
					tempX := len(env.CurPanel.Rows[env.CurPanel.CursorPos.Y-1])
					env.CurPanel.Rows[env.CurPanel.CursorPos.Y-1] += row
					env.CurPanel.Rows = append(env.CurPanel.Rows[:env.CurPanel.CursorPos.Y], env.CurPanel.Rows[env.CurPanel.CursorPos.Y+1:]...)

					env.CurPanel.CursorPos.Y--
					env.CurPanel.CursorPos.X = tempX

				} else {
					row := env.CurPanel.Rows[env.CurPanel.CursorPos.Y]
					row = string(row[:env.CurPanel.CursorPos.X-1]) + string(row[env.CurPanel.CursorPos.X:])
					env.CurPanel.Rows[env.CurPanel.CursorPos.Y] = row
					env.CurPanel.CursorPos.X--

				}

			case 13:
				row := env.CurPanel.Rows[env.CurPanel.CursorPos.Y]
				newLine := row[env.CurPanel.CursorPos.X:]
				row = row[:env.CurPanel.CursorPos.X]
				env.CurPanel.Rows[env.CurPanel.CursorPos.Y] = newLine
				env.CurPanel.Rows = append(env.CurPanel.Rows[:env.CurPanel.CursorPos.Y], append([]string{row}, env.CurPanel.Rows[env.CurPanel.CursorPos.Y:]...)...)
				env.CurPanel.CursorPos.Y++
				env.CurPanel.CursorPos.X = 0

			default:
				row := env.CurPanel.Rows[env.CurPanel.CursorPos.Y]
				row = row[:env.CurPanel.CursorPos.X] + string(c) + row[env.CurPanel.CursorPos.X:]
				env.CurPanel.Rows[env.CurPanel.CursorPos.Y] = row
				env.CurPanel.CursorPos.X++
			}
		}
	}
	return nil
}

func getWindowSize(fd uintptr) (*unix.Winsize, error) {
	winSize, err := unix.IoctlGetWinsize(int(fd), ioctlGetWindowSize)
	if err != nil {
		return nil, err
	}
	return winSize, nil
}

//func getCursorPosition(fd uintptr) (CursorPosition, error) {
//}

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
