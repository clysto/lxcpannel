package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/canonical/lxd/shared/api"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type CommandFunc func(sess ssh.Session, args []string) int

var commands = map[string]CommandFunc{
	"pubkey": pubkeyCmd,
	"whoami": whoamiCmd,
	"lxc":    lxcCmd,
	"ls":     lsCmd,
	"ssh":    sshCmd,
	"ip":     ipCmd,
}

func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			cmd.Usage()
			cmd.Println()
			return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

func ipCmd(sess ssh.Session, args []string) int {
	addr := sess.LocalAddr()
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		wish.Printf(sess, "Error: %v\n", err)
		return 1
	}
	wish.Printf(sess, "%s\n", host)
	return 0
}

func pubkeyCmd(sess ssh.Session, args []string) int {
	cmd := cobra.Command{
		Use: "pubkey",
	}
	cmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := listPubkeys(sess.User())
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
		Use:  "add <public key>",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := addPubkey(sess.User(), args[0])
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "delete <fingerprint>",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := deletePubkey(sess.User(), args[0])
			return err
		},
	})
	cmd.SetArgs(args[1:])
	cmd.SetIn(sess)
	cmd.SetOut(sess)
	cmd.SetErr(sess)
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err != nil {
		return 1
	}
	return 0
}

func whoamiCmd(sess ssh.Session, args []string) int {
	user, err := getUser(sess.User())
	if err != nil {
		return 1
	}
	wish.Printf(sess, "username: %s\n", user.Username)
	wish.Printf(sess, "admin: %t\n", user.Admin)
	wish.Printf(sess, "max_instance_count: %d\n", user.MaxInstanceCount)
	return 0
}

func lsCmd(sess ssh.Session, args []string) int {
	newArgs := append([]string{"lxc", "list"}, args[1:]...)
	return lxcCmd(sess, newArgs)
}

func sshCmd(sess ssh.Session, args []string) int {
	newArgs := append([]string{"lxc", "shell"}, args[1:]...)
	return lxcCmd(sess, newArgs)
}

func lxcCmd(sess ssh.Session, args []string) int {
	cmd := cobra.Command{
		Use: "lxc",
	}
	cmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			containers, err := client.ListContainers(sess.User())
			if err != nil {
				return err
			}
			table := tablewriter.NewWriter(sess)
			table.SetRowLine(true)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"Name", "Friendly Name", "State", "SSH Port"})
			for _, container := range containers {
				port := client.SSHPort(container.Name)
				name := container.Config["user.friendlyname"]
				portStr := ""
				if port == 0 {
					portStr = "N/A"
				} else {
					portStr = strconv.Itoa(port)
				}
				table.Append([]string{container.Name, name, container.Status, portStr})
			}
			table.Render()
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "start <name>",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := client.StartContainer(sess.User(), args[0])
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "stop <name>",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := client.StopContainer(sess.User(), args[0])
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "delete <name>",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := client.DeleteContainer(sess.User(), args[0])
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "info <name>",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			container, err := client.GetContainer(sess.User(), args[0])
			if err != nil {
				return err
			}
			yaml.NewEncoder(sess).Encode(container)
			return nil
		},
	})
	createCmd := &cobra.Command{
		Use:  "create <friendly name>",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containers, err := client.ListContainers(sess.User())
			if err != nil {
				return err
			}
			user, err := getUser(sess.User())
			if err != nil {
				return err
			}
			if len(containers) >= user.MaxInstanceCount {
				return errors.New("max instance count reached")
			}
			progress := ProgressRenderer{
				sess: sess,
			}
			image, err := cmd.Flags().GetString("fingerprint")
			if err != nil {
				return err
			}
			op, err := client.CreateContainer(sess.User(), args[0], image)
			if err != nil {
				return err
			}
			op.AddHandler(progress.UpdateOp)
			err = op.Wait()
			return err
		},
	}
	createCmd.Flags().String("fingerprint", client.defaultImage, "image fingerprint")
	cmd.AddCommand(createCmd)
	cmd.AddCommand(&cobra.Command{
		Use:  "shell <name>",
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			container, err := client.GetContainer(sess.User(), args[0])
			if err != nil {
				return err
			}
			ch := make(chan api.InstanceExecControl)
			_, windowChanges, _ := sess.Pty()

			go func() {
				for {
					select {
					case window, ok := <-windowChanges:
						if !ok {
							return
						}
						ch <- api.InstanceExecControl{
							Command: "window-resize",
							Args: map[string]string{
								"width":  strconv.Itoa(window.Width),
								"height": strconv.Itoa(window.Height),
							},
						}
					case <-ctx.Done():
						return
					}
				}
			}()
			err = client.StartShell(container.Name, cmd.InOrStdin(), cmd.OutOrStdout(), ch)
			if err != nil {
				return err
			}
			sess.(*SessionWrapper).DummyWrite()
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use: "images",
		RunE: func(cmd *cobra.Command, args []string) error {
			images, err := client.ListImages()
			if err != nil {
				return err
			}
			table := tablewriter.NewWriter(sess)
			table.SetRowLine(true)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"Fingerprint", "Description", "Type", "Size"})
			for _, image := range images {
				table.Append([]string{
					image.Fingerprint[:16],
					image.Properties["description"],
					image.Type,
					fmt.Sprintf("%dMB", image.Size/1024/1024),
				})
			}
			table.Render()
			return nil
		},
	})
	cmd.SetArgs(args[1:])
	cmd.SetIn(sess)
	cmd.SetOut(sess)
	cmd.SetErr(sess)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	if err != nil {
		wish.Printf(sess, "Error: %v\n", err)
		return 1
	}
	return 0
}
