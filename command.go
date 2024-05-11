package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/canonical/lxd/shared/api"
	"github.com/charmbracelet/ssh"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type CommandFunc func(ctx *CommandContext, args []string) int

var Commands = map[string]CommandFunc{
	"pubkey": pubkeyCmd,
	"whoami": whoamiCmd,
	"lxc":    lxcCmd,
	"ls":     lsCmd,
	"ssh":    sshCmd,
	"ip":     ipCmd,
}

func ipCmd(ctx *CommandContext, args []string) int {
	fmt.Fprintf(ctx, "%s\n", ctx.IP())
	return 0
}

func pubkeyCmd(ctx *CommandContext, args []string) int {
	cmd := cobra.Command{
		Use: "pubkey",
	}
	cmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := ListPubkeys(ctx.User())
			if err != nil {
				return err
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Fingerprint", "Public Key"})
			table.SetRowLine(true)
			for _, dbkey := range keys {
				table.Append([]string{dbkey.Fingerprint[:16], WordWrap(dbkey.PEM, 48)})
			}
			table.Render()
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "add <public key>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return AddPubkey(ctx.User(), args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "delete <fingerprint>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := DeletePubkey(ctx.User(), args[0])
			return err
		},
	})
	cmd.SetArgs(args[1:])
	cmd.SetIn(ctx)
	cmd.SetOut(ctx)
	cmd.SetErr(ctx)
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err != nil {
		return 1
	}
	return 0
}

func whoamiCmd(ctx *CommandContext, args []string) int {
	user, err := GetUser(ctx.User())
	if err != nil {
		return 1
	}
	table := tablewriter.NewWriter(ctx)
	table.SetHeader([]string{"Username", "Admin", "Max Instance Count"})
	table.SetRowLine(true)
	table.Append([]string{user.Username, fmt.Sprintf("%t", user.Admin), strconv.Itoa(user.MaxInstanceCount)})
	table.Render()
	return 0
}

func lsCmd(ctx *CommandContext, args []string) int {
	newArgs := append([]string{"lxc", "list"}, args[1:]...)
	return lxcCmd(ctx, newArgs)
}

func sshCmd(ctx *CommandContext, args []string) int {
	newArgs := append([]string{"lxc", "shell"}, args[1:]...)
	return lxcCmd(ctx, newArgs)
}

func lxcCmd(ctx *CommandContext, args []string) int {
	cmd := cobra.Command{
		Use: "lxc",
	}
	cmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			containers, err := client.ListContainers(ctx.User())
			if err != nil {
				return err
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
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
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := client.StartContainer(ctx.User(), args[0])
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "stop <name>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := client.StopContainer(ctx.User(), args[0])
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "delete <name>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := client.DeleteContainer(ctx.User(), args[0])
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "info <name>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			container, err := client.GetContainer(ctx.User(), args[0])
			if err != nil {
				return err
			}
			yaml.NewEncoder(cmd.OutOrStdout()).Encode(container)
			return nil
		},
	})
	createCmd := &cobra.Command{
		Use:  "create <friendly name>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containers, err := client.ListContainers(ctx.User())
			if err != nil {
				return err
			}
			user, err := GetUser(ctx.User())
			if err != nil {
				return err
			}
			if len(containers) >= user.MaxInstanceCount {
				return errors.New("max instance count reached")
			}
			progress := ProgressRenderer{
				out: ctx,
			}
			image, err := cmd.Flags().GetString("fingerprint")
			if err != nil {
				return err
			}
			op, err := client.CreateContainer(ctx.User(), args[0], image)
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
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			container, err := client.GetContainer(ctx.User(), args[0])
			if err != nil {
				return err
			}
			ch := make(chan api.InstanceExecControl)

			id := ctx.OnWindowChange(func(window ssh.Window) {
				ch <- api.InstanceExecControl{
					Command: "window-resize",
					Args: map[string]string{
						"width":  strconv.Itoa(window.Width),
						"height": strconv.Itoa(window.Height),
					},
				}
			})
			width, height := ctx.WindowSize()
			err = client.StartShell(container.Name, cmd.InOrStdin(), cmd.OutOrStdout(), width, height, ch)
			if err != nil {
				return err
			}
			ctx.StdinWrite([]byte(" "))
			ctx.RemoveWindowChangeHandler(id)
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
			table := tablewriter.NewWriter(cmd.OutOrStdout())
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
	cmd.SetIn(ctx)
	cmd.SetOut(ctx)
	cmd.SetErr(ctx)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(ctx, "Error: %v\n", err)
		return 1
	}
	return 0
}

func CommandComplete(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
	if key == '\t' {
		line = strings.TrimLeft(line, " ")
		pos -= len(line) - len(strings.TrimLeft(line, " "))

		firstSpace := strings.Index(line, " ")
		var prefix string
		if firstSpace == -1 {
			prefix = line
		} else {
			prefix = line[:firstSpace]
		}

		for command := range Commands {
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
