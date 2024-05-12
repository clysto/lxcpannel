package cmd

import "fmt"

type ipCmd struct{}

func (cmd *ipCmd) Exec(ctx *CommandContext, args []string) error {
	fmt.Fprintf(ctx, "%s\n", ctx.IP())
	return nil
}
