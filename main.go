package main

import (
	"flag"
	"fmt"
	"net"
	"strings"

	_ "embed"

	"github.com/anmitsu/go-shlex"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
	"golang.org/x/term"
)

var client *LXCClient

//go:embed banner.txt
var banner string

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
	InitDB(*dbPath)
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", *port))),
		wish.WithHostKeyPath(*keyPath),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			user := ctx.User()
			keys, err := ListPubkeys(user)
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
					ctx := &CommandContext{
						sess: sess,
						ch:   make(chan []byte, 1024),
					}
					ctx.Connect()

					ip := ctx.IP()
					prompt := "\033[01;32m" + sess.User() + "@" + ip + "\033[0m:\033[01;34mustc\033[0m$ "

					terminal := term.NewTerminal(ctx, prompt)
					terminal.AutoCompleteCallback = CommandComplete
					id := ctx.OnWindowChange(func(window ssh.Window) {
						terminal.SetSize(window.Width, window.Height)
					})

					fmt.Fprint(terminal, banner)
					fmt.Fprintf(terminal, "IPv4 address: %s\n", ip)

					for {
						line, err := terminal.ReadLine()
						if err != nil {
							break
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
						f := Commands[args[0]]
						if f != nil {
							f(ctx, args)
						} else {
							fmt.Fprintf(terminal, "%s: command not found\n", args[0])
						}
					}

					ctx.RemoveWindowChangeHandler(id)
					ctx.Disconnect()
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
