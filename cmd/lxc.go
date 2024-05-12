package cmd

import (
	"errors"
	"fmt"
	"lxcpanel/common"
	"strconv"

	"github.com/canonical/lxd/shared/api"
	"github.com/charmbracelet/ssh"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type lxcCmd struct {
	cmd cobra.Command
	ctx *CommandContext
}

func (command *lxcCmd) Exec(ctx *CommandContext, args []string) error {
	command.cmd.SetArgs(args[1:])
	command.cmd.SetIn(ctx)
	command.cmd.SetOut(ctx)
	command.cmd.SetErr(ctx)
	command.ctx = ctx
	return command.cmd.Execute()
}

func NewLxcCmd() Command {
	command := &lxcCmd{
		cmd: cobra.Command{
			Use: "lxc",
		},
		ctx: nil,
	}
	command.cmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := command.ctx
			containers, err := common.LxcClient.ListContainers(ctx.User())
			if err != nil {
				return err
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetRowLine(true)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"Name", "Friendly Name", "State", "SSH Port"})
			for _, container := range containers {
				port := common.LxcClient.SSHPort(container.Name)
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
	command.cmd.AddCommand(&cobra.Command{
		Use:  "start <name>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := command.ctx
			err := common.LxcClient.StartContainer(ctx.User(), args[0])
			return err
		},
	})
	command.cmd.AddCommand(&cobra.Command{
		Use:  "stop <name>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := command.ctx
			err := common.LxcClient.StopContainer(ctx.User(), args[0])
			return err
		},
	})
	command.cmd.AddCommand(&cobra.Command{
		Use:  "delete <name>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := command.ctx
			err := common.LxcClient.DeleteContainer(ctx.User(), args[0])
			return err
		},
	})
	command.cmd.AddCommand(&cobra.Command{
		Use:  "info <name>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := command.ctx
			container, err := common.LxcClient.GetContainer(ctx.User(), args[0])
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
			ctx := command.ctx
			containers, err := common.LxcClient.ListContainers(ctx.User())
			if err != nil {
				return err
			}
			user, err := common.GetUser(ctx.User())
			if err != nil {
				return err
			}
			if len(containers) >= user.MaxInstanceCount {
				return errors.New("max instance count reached")
			}
			progress := common.NewProgressRenderer(ctx)
			image, err := cmd.Flags().GetString("fingerprint")
			if err != nil {
				return err
			}
			op, err := common.LxcClient.CreateContainer(ctx.User(), args[0], image)
			if err != nil {
				return err
			}
			op.AddHandler(progress.UpdateOp)
			err = op.Wait()
			return err
		},
	}
	createCmd.Flags().String("fingerprint", common.LxcClient.DefaultImage(), "image fingerprint")
	command.cmd.AddCommand(createCmd)
	command.cmd.AddCommand(&cobra.Command{
		Use:  "shell <name>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := command.ctx
			container, err := common.LxcClient.GetContainer(ctx.User(), args[0])
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
			err = common.LxcClient.StartShell(container.Name, cmd.InOrStdin(), cmd.OutOrStdout(), width, height, ch)
			if err != nil {
				return err
			}
			ctx.SendEOF()
			ctx.RemoveWindowChangeHandler(id)
			return nil
		},
	})
	command.cmd.AddCommand(&cobra.Command{
		Use: "images",
		RunE: func(cmd *cobra.Command, args []string) error {
			images, err := common.LxcClient.ListImages()
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
	command.cmd.SilenceUsage = true
	command.cmd.SilenceErrors = true
	return command
}
