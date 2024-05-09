package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/anmitsu/go-shlex"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
	"github.com/chzyer/readline"
)

const (
	host = "localhost"
	port = "23235"
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithBanner("Welcome to the LXC Pannel\n"),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			user := ctx.User()
			keys, err := list_pubkeys(user)
			if err != nil {
				return false
			}
			for _, dbkey := range keys {
				upk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(dbkey.PEM))
				if err != nil {
					return false
				}
				if ssh.KeysEqual(key, upk) {
					return true
				}
			}
			return false
		}),
		wish.WithMiddleware(
			func(next ssh.Handler) ssh.Handler {
				return func(sess ssh.Session) {
					pcitems := []readline.PrefixCompleterInterface{}
					for k := range commands {
						pcitems = append(pcitems, readline.PcItem(k))
					}
					l, err := readline.NewEx(&readline.Config{
						Prompt:          "\033[01;32m" + sess.User() + "@lxc\033[0m$ ",
						InterruptPrompt: "^C",
						EOFPrompt:       "exit",
						AutoComplete: readline.NewPrefixCompleter(
							pcitems...,
						),
						HistorySearchFold: true,
						Stdin:             sess,
						StdinWriter:       sess,
						Stdout:            sess,
						Stderr:            sess,
					})
					if err != nil {
						log.Error("Could not create readline", "error", err)
						return
					}
					defer l.Close()
					for {
						line, err := l.Readline()
						if err != nil {
							if err != io.EOF {
								log.Error("Could not read line", "error", err)
							}
							return
						}
						line = strings.TrimSpace(line)
						if line == "exit" {
							break
						}
						if line == "" {
							continue
						}
						args, _ := shlex.Split(line, true)
						args = append([]string(nil), args...)
						f := commands[args[0]]
						if f != nil {
							f(sess, args)
						} else {
							sess.Write([]byte(fmt.Sprintf("%s: command not found\n", args[0])))
						}
					}
				}
			},
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}
