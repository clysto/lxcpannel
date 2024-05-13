package cmd

import (
	"fmt"
	"lxcpanel/common"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type adminCmd struct {
	cmd cobra.Command
	ctx *CommandContext
}

func (command *adminCmd) Exec(ctx *CommandContext, args []string) error {
	command.cmd.SetArgs(args[1:])
	command.cmd.SetIn(ctx)
	command.cmd.SetOut(ctx)
	command.cmd.SetErr(ctx)
	command.ctx = ctx
	return command.cmd.Execute()
}

func NewAdminCmd() Command {
	command := &adminCmd{
		cmd: cobra.Command{
			Use: "admin",
		},
		ctx: nil,
	}
	userCmd := &cobra.Command{
		Use: "user",
	}
	command.cmd.AddCommand(userCmd)
	userCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			users, err := common.ListUsers()
			if err != nil {
				return err
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetRowLine(true)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"Username", "Admin", "Max Instance Count"})
			for _, user := range users {
				table.Append([]string{user.Username, fmt.Sprintf("%t", user.Admin), strconv.Itoa(user.MaxInstanceCount)})
			}
			table.Render()
			return nil
		},
	})
	userAddCmd := &cobra.Command{
		Use:  "add <username>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			maxInstanceCount, err := cmd.Flags().GetInt("max-instance-count")
			if err != nil {
				return err
			}
			admin := cmd.Flags().Changed("admin")
			return common.AddUser(args[0], admin, maxInstanceCount)
		},
	}
	userAddCmd.Flags().Bool("admin", false, "Make the user an admin")
	userAddCmd.Flags().IntP("max-instance-count", "n", 3, "The maximum number of instances the user can create")
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(&cobra.Command{
		Use:  "delete <username>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.DeleteUser(args[0])
		},
	})
	userCmd.AddCommand(&cobra.Command{
		Use:  "instances <username> <num>",
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			maxInstanceCount, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			return common.ChangeMaxInstanceCount(args[0], maxInstanceCount)
		},
	})
	pubkeyCmd := &cobra.Command{
		Use: "pubkey",
	}
	command.cmd.AddCommand(pubkeyCmd)
	pubkeyCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			pubkeys, err := common.ListAllPubkeys()
			if err != nil {
				return err
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetRowLine(true)
			table.SetHeader([]string{"Fingerprint", "Username"})
			for _, pubkey := range pubkeys {
				table.Append([]string{pubkey.Fingerprint[:16], pubkey.Username})
			}
			table.Render()
			return nil
		},
	})

	pubkeyCmd.AddCommand(&cobra.Command{
		Use:  "show <fingerprint>",
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubkey, err := common.GetPubkey(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Username: %s\n", pubkey.Username)
			fmt.Fprintf(cmd.OutOrStdout(), "Fingerprint: %s\n", pubkey.Fingerprint)
			fmt.Fprintln(cmd.OutOrStdout(), "Public key:")
			fmt.Fprintln(cmd.OutOrStdout(), pubkey.PEM)
			return nil
		},
	})

	pubkeyCmd.AddCommand(&cobra.Command{
		Use:  "delete <username> <fingerprint>",
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return common.DeletePubkey(args[0], args[1])
		},
	})

	pubkeyCmd.AddCommand(&cobra.Command{
		Use:  "add <username> <pubkey>",
		Args: MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.Join(args[1:], " ")
			return common.AddPubkey(args[0], key)
		},
	})

	command.cmd.SilenceErrors = true
	command.cmd.SilenceUsage = true
	return command
}
