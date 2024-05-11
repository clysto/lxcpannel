package main

import (
	"fmt"
	"io"
	"net"

	"github.com/charmbracelet/ssh"
	"github.com/spf13/cobra"
)

type CommandContext struct {
	sess ssh.Session
	ch   chan []byte
}

func (s *CommandContext) Connect() {
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

func (s *CommandContext) WindowChanges() <-chan ssh.Window {
	_, windowChanges, _ := s.sess.Pty()
	return windowChanges
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
