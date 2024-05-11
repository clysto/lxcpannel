package main

import (
	"fmt"
	"io"
	"net"

	"github.com/charmbracelet/ssh"
	"github.com/spf13/cobra"
)

type CommandContext struct {
	sess                ssh.Session
	ch                  chan []byte
	windowChangeHanders []func(ssh.Window)
	done                chan bool
	windowWidth         int
	windowHeight        int
}

func (s *CommandContext) Connect() {
	s.done = make(chan bool)
	go func() {
		var buffer [1024]byte
		for {
			n, err := s.sess.Read(buffer[:])
			if err != nil {
				close(s.ch)
				return
			}
			if n > 0 {
				s.ch <- buffer[:n]
			}
		}
	}()
	_, windowChanges, _ := s.sess.Pty()
	go func() {
		for {
			select {
			case <-s.done:
				return
			case window := <-windowChanges:
				for _, f := range s.windowChangeHanders {
					s.windowWidth = window.Width
					s.windowHeight = window.Height
					f(window)
				}
			}
		}
	}()
}

func (s *CommandContext) Disconnect() {
	s.done <- true
}

func (s *CommandContext) OnWindowChange(f func(ssh.Window)) int {
	s.windowChangeHanders = append(s.windowChangeHanders, f)
	return len(s.windowChangeHanders) - 1
}

func (s *CommandContext) WindowSize() (int, int) {
	return s.windowWidth, s.windowHeight
}

func (s *CommandContext) RemoveWindowChangeHandler(id int) {
	s.windowChangeHanders = append(s.windowChangeHanders[:id], s.windowChangeHanders[id+1:]...)
}

func (s *CommandContext) StdinWrite(p []byte) {
	s.ch <- p
}

func (s *CommandContext) Read(p []byte) (n int, err error) {
	data, ok := <-s.ch
	if !ok {
		return 0, io.EOF
	}
	n = copy(p, data)
	return n, nil
}

func (s *CommandContext) Write(p []byte) (n int, err error) {
	return s.sess.Write(p)
}

func (s *CommandContext) IP() string {
	addr := s.sess.LocalAddr()
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return ""
	}
	return host
}

func (s *CommandContext) User() string {
	return s.sess.User()
}

func WordWrap(text string, lineWidth int) string {
	var result string

	for i := 0; i < len(text); i += lineWidth {
		if i+lineWidth > len(text) {
			result += text[i:]
		} else {
			result += text[i:i+lineWidth] + "\n"
		}
	}

	return result
}

func ExactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			cmd.Usage()
			cmd.Println()
			return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}
