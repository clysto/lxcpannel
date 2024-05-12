package cmd

import (
	"lxcpanel/common"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type pubkeyCmd struct {
	cmd cobra.Command
	ctx *CommandContext
}

func (command *pubkeyCmd) Exec(ctx *CommandContext, args []string) error {
	command.cmd.SetArgs(args[1:])
	command.cmd.SetIn(ctx)
	command.cmd.SetOut(ctx)
	command.cmd.SetErr(ctx)
	command.ctx = ctx
	return command.cmd.Execute()
}

func NewPubkeyCmd() Command {
	command := &pubkeyCmd{
		cmd: cobra.Command{
			Use: "pubkey",
		},
		ctx: nil,
	}
	command.cmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := command.ctx
			keys, err := common.ListPubkeys(ctx.User())
			if err != nil {
				return err
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Fingerprint", "Public Key"})
			table.SetRowLine(true)
			for _, dbkey := range keys {
				table.Append([]string{dbkey.Fingerprint[:16], common.WordWrap(dbkey.PEM, 48)})
			}
			table.Render()
			return nil
		},
	})
	command.cmd.AddCommand(&cobra.Command{
		Use:  "add <public key>",
		Args: MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := command.ctx
			key := strings.Join(args, " ")
			return common.AddPubkey(ctx.User(), key)
		},
	})
	command.cmd.AddCommand(&cobra.Command{
		Use:  "delete <fingerprint>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := command.ctx
			err := common.DeleteUserPubkey(ctx.User(), args[0])
			return err
		},
	})
	command.cmd.SilenceUsage = true
	command.cmd.SilenceErrors = true
	return command
}
