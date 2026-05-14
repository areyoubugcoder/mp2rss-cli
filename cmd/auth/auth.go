// Package auth implements `mp2rss auth ...` subcommands.
package auth

import (
	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/spf13/cobra"
)

// NewCmd builds the auth command tree.
func NewCmd(deps *cliopts.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "管理 Feed Key 认证（登录 / 登出 / 状态）",
	}
	cmd.AddCommand(newLoginCmd(deps))
	cmd.AddCommand(newLogoutCmd(deps))
	cmd.AddCommand(newStatusCmd(deps))
	return cmd
}
