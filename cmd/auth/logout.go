package auth

import (
	"fmt"

	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/config"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/spf13/cobra"
)

func newLogoutCmd(deps *cliopts.Deps) *cobra.Command {
	_ = deps
	return &cobra.Command{
		Use:   "logout",
		Short: "清除本地 Feed Key（保留 api_url）",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := config.Get()
			if err := cfg.Clear(); err != nil {
				return errs.Wrap(errs.CodeGeneric, err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ 已退出登录。")
			return nil
		},
	}
}
