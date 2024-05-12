package main

import (
	"flag"
	"fmt"
	"lxcpanel/cmd"
	"lxcpanel/common"
	"lxcpanel/lxc"
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

//go:embed banner.txt
var banner string

func main() {
	port := flag.Int("port", 2222, "port to listen on")
	profile := flag.String("profile", "default", "LXD profile to use")
	dbPath := flag.String("db", "lxcpanel.sqlite3", "path to database")
	keyPath := flag.String("key", ".ssh/id_ed25519", "path to host key")
	defaultImage := flag.String("image", "c9fba5728bfe168a", "default image to use")
	host := flag.String("host", "0.0.0.0", "host to listen on")
	flag.Parse()
	var err error
	common.Client, err = lxc.NewLXCClient(*profile, *defaultImage)
	if err != nil {
		panic(err)
	}
	common.InitDB(*dbPath)
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(*host, fmt.Sprintf("%d", *port))),
		wish.WithHostKeyPath(*keyPath),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			user := ctx.User()
			keys, err := common.ListPubkeys(user)
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
					ctx := cmd.NewCommandContext(sess)

					ip := ctx.IP()
					prompt := "\033[01;32m" + sess.User() + "@" + ip + "\033[0m:\033[01;34mustc\033[0m$ "

					user, err := common.GetUser(sess.User())
					if err != nil {
						log.Error("Error getting user", "error", err)
						next(sess)
						return
					}
					commands := cmd.BuildCmdList(user.Admin)
					terminal := term.NewTerminal(ctx, prompt)
					terminal.SetSize(ctx.WindowSize())
					terminal.AutoCompleteCallback = cmd.BuildCompletionFunc(commands)
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
						cmd := commands[args[0]]
						if cmd != nil {
							err := cmd.Exec(ctx, args)
							if err != nil {
								fmt.Fprintf(terminal, "Error: %s\n", err)
							}
						} else {
							fmt.Fprintf(terminal, "%s: command not found\n", args[0])
						}
					}

					ctx.RemoveWindowChangeHandler(id)
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
