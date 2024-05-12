package cmd

import (
	"fmt"
	"lxcpanel/common"
	"strconv"

	"github.com/olekukonko/tablewriter"
)

type whoamiCmd struct{}

func (cmd *whoamiCmd) Exec(ctx *CommandContext, args []string) error {
	user, err := common.GetUser(ctx.User())
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(ctx)
	table.SetHeader([]string{"Username", "Admin", "Max Instance Count"})
	table.SetRowLine(true)
	table.Append([]string{user.Username, fmt.Sprintf("%t", user.Admin), strconv.Itoa(user.MaxInstanceCount)})
	table.Render()
	return nil
}
