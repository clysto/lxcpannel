package main

import (
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/spf13/cobra"
)

type CommandFunc func(sess ssh.Session, args []string) int

var commands = map[string]CommandFunc{
	"pubkey": pubkey,
	"whoami": whoami,
}

func pubkey(sess ssh.Session, args []string) int {
	cmd := cobra.Command{
		Use: "pubkey",
	}
	cmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := list_pubkeys(sess.User())
			if err != nil {
				return err
			}
			for i, dbkey := range keys {
				sess.Write([]byte(dbkey.Fingerprint[:16] + ": "))
				sess.Write([]byte(dbkey.PEM + "\n"))
				if i != len(keys)-1 {
					sess.Write([]byte("\n"))
				}
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "add",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := add_pubkey(sess.User(), args[0])
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "delete",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := delete_pubkey(sess.User(), args[0])
			return err
		},
	})
	cmd.SetArgs(args[1:])
	cmd.SetIn(sess)
	cmd.SetOut(sess)
	cmd.SetErr(sess)
	err := cmd.Execute()
	if err != nil {
		return 1
	}
	return 0
}

func whoami(sess ssh.Session, args []string) int {
	user, err := show_user(sess.User())
	if err != nil {
		return 1
	}
	wish.Printf(sess, "Username: %s\n", user.Username)
	wish.Printf(sess, "Admin: %t\n", user.Admin)
	wish.Printf(sess, "MaxInstanceCount: %d\n", user.MaxInstanceCount)
	return 0
}
