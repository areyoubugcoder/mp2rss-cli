// Package update is a placeholder for the self-update command.
//
// Real implementation lands in Phase 2 (see plan). Phase 1 ships a stub so
// the binary already advertises the command surface.
package update

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewCmd returns the `mp2rss update` placeholder command.
func NewCmd() *cobra.Command {
	var (
		flagForce bool
		flagCheck bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "自更新 CLI 二进制（v1.x 启用，当前为占位命令）",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "ℹ 自更新功能将在 v1.x 版本启用。")
			fmt.Fprintln(cmd.OutOrStdout(), "  当前请通过 npm / Homebrew / 直接下载 release 升级。")
			return nil
		},
	}
	cmd.Flags().BoolVar(&flagForce, "force", false, "强制下载最新版本（占位）")
	cmd.Flags().BoolVar(&flagCheck, "check", false, "仅检查是否有新版本（占位）")
	return cmd
}
