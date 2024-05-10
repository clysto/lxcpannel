package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"strings"

	_ "embed"

	"github.com/anmitsu/go-shlex"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
	"github.com/chzyer/readline"
)

var client *LXCClient

//go:embed banner.txt
var banner string

type SessionWrapper struct {
	ssh.Session
	ch chan []byte
}

func (s *SessionWrapper) Connect() {
	go func() {
		var buffer [1024]byte
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
	dbPath := flag.String("db", "lxcpanel.sqlite3", "path to database")
	keyPath := flag.String("key", ".ssh/id_ed25519", "path to host key")
	defaultImage := flag.String("image", "c9fba5728bfe168a", "default image to use")
	flag.Parse()
	var err error
	client, err = NewLXCClient(*profile, *defaultImage)
	if err != nil {
		panic(err)
	}
	initDB(*dbPath)
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", *port))),
		wish.WithHostKeyPath(*keyPath),
		wish.WithBanner(banner),
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
						wish.Printf(sess, "IPv4 address: %s\n", host)
					}
					pcitems := []readline.PrefixCompleterInterface{}
					for k := range commands {
						pcitems = append(pcitems, readline.PcItem(k))
					}
					rl, err := readline.NewEx(&readline.Config{
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
						// ForceUseInteractive: true,
					})
					if err != nil {
						log.Error("Could not create readline", "error", err)
						return
					}
					defer rl.Close()
					for {
						line, err := rl.Readline()
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
					next(sess)
				}
			},
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
