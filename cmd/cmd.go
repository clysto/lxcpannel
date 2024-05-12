package cmd

import (
	"fmt"
	"lxcpanel/common"
	"net"
	"strings"

	"github.com/charmbracelet/ssh"
	"github.com/spf13/cobra"
)

type Command interface {
	Exec(ctx *CommandContext, args []string) error
}

type AliasCommand struct {
	Cmd  Command
	Args []string
}

type CommandContext struct {
	sess                ssh.Session
	windowChangeHanders []func(ssh.Window)
	windowWidth         int
	windowHeight        int
	reader              *common.InterruptibleReader
}

func NewCommandContext(sess ssh.Session) *CommandContext {
	ctx := &CommandContext{
		sess:   sess,
		reader: common.NewInterruptibleReader(sess),
	}
	go ctx.monitorWindow()
	return ctx
}

func (s *CommandContext) monitorWindow() {
	_, windowChanges, _ := s.sess.Pty()
	for {
		window, ok := <-windowChanges
		if !ok {
			return
		}
		for _, f := range s.windowChangeHanders {
			s.windowWidth = window.Width
			s.windowHeight = window.Height
			f(window)
		}
	}
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

func (s *CommandContext) SendEOF() {
	s.reader.SendEOF()
}

func (s *CommandContext) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
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

func MinimumNArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < n {
			cmd.Usage()
			cmd.Println()
			return fmt.Errorf("requires at least %d arg(s), only received %d", n, len(args))
		}
		return nil
	}
}

func BuildCmdList() map[string]Command {
	lxc := NewLxcCmd()
	return map[string]Command{
		"pubkey": NewPubkeyCmd(),
		"lxc":    lxc,
		"ip":     &ipCmd{},
		"whoami": &whoamiCmd{},
		"ls": &AliasCommand{
			Cmd:  lxc,
			Args: []string{"list"},
		},
		"ssh": &AliasCommand{
			Cmd:  lxc,
			Args: []string{"shell"},
		},
	}
}

func BuildCompletionFunc(commands map[string]Command) func(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
	return func(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
		if key == '\t' && pos > 0 {
			line = strings.TrimLeft(line, " ")
			pos -= len(line) - len(strings.TrimLeft(line, " "))

			firstSpace := strings.Index(line, " ")
			var prefix string
			if firstSpace == -1 {
				prefix = line
			} else {
				prefix = line[:firstSpace]
			}

			for command := range commands {
				if strings.HasPrefix(command, prefix) {
					if firstSpace == -1 {
						return command, len(command), true
					} else {
						return command + line[firstSpace:], pos, true
					}
				}
			}
		}

		return line, pos, false
	}
}

func (command *AliasCommand) Exec(ctx *CommandContext, args []string) error {
	newArgs := make([]string, 0)
	newArgs = append(newArgs, args[0])
	newArgs = append(newArgs, command.Args...)
	newArgs = append(newArgs, args[1:]...)
	return command.Cmd.Exec(ctx, newArgs)
}
