package main

import (
	"flag"
	"fmt"
	"io"
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

type SessionWrapper struct {
	ssh.Session
	ch chan []byte
}

func (s *SessionWrapper) Connect() {
	go func() {
		var buffer [32 * 1024]byte
		for {
			n, err := s.Session.Read(buffer[:])
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

func (s *SessionWrapper) DummyWrite() {
	s.ch <- []byte(" ")
}

func (s *SessionWrapper) Read(p []byte) (n int, err error) {
	data, ok := <-s.ch
	if !ok {
		return 0, io.EOF
	}
	n = copy(p, data)
	return n, nil
}

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
				return func(sess_ ssh.Session) {
					sess := &SessionWrapper{
						Session: sess_,
						ch:      make(chan []byte, 1024),
					}
					sess.Connect()

					addr := sess.LocalAddr()
					host, _, err := net.SplitHostPort(addr.String())
					if err == nil {
						wish.Printf(sess, "IP: %s\n", host)
					}

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
							f(sess, l, args)
						} else {
							sess.Write([]byte(fmt.Sprintf("%s: command not found\n", args[0])))
						}
					}
					next(sess)
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
