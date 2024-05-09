package main

import (
	"flag"
	"fmt"
	"net"
	"strings"

	"github.com/anmitsu/go-shlex"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/logging"
	"github.com/chzyer/readline"
)

var client *LXCClient

func main() {
	port := flag.Int("port", 2222, "port to listen on")
	profile := flag.String("profile", "default", "LXD profile to use")
	flag.Parse()
	var err error
	client, err = NewLXCClient(*profile)
	if err != nil {
		panic(err)
	}
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", *port))),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithBanner("Welcome to the LXC Pannel\n"),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			user := ctx.User()
			keys, err := listPubkeys(user)
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
						Prompt:          "\033[01;32m" + sess.User() + "@lxcpanel\033[0m$ ",
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
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		panic(err)
	}

	log.Info("Starting SSH server", "host", "0.0.0.0", "port", *port)
	if err = s.ListenAndServe(); err != nil {
		panic(err)
	}
}
