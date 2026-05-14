package mp

import (
	"fmt"

	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/spf13/cobra"
)

func newSearchCmd(deps *cliopts.Deps) *cobra.Command {
	var (
		flagPage     int
		flagPageSize int
	)

	cmd := &cobra.Command{
		Use:   "search <keyword>",
		Short: "按公众号名称搜索（mp list -q 的语法糖）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kw := args[0]
			if kw == "" {
				return errs.Newf(errs.CodeArgs, "关键词不能为空")
			}
			c, err := newAuthedClient()
			if err != nil {
				return err
			}
			list, err := c.ListSubscriptions(kw, flagPage, flagPageSize)
			if err != nil {
				return err
			}
			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), list)
			}
			renderSubscriptionTable(cmd, list.Items)
			fmt.Fprintf(cmd.OutOrStdout(), "\n关键词「%s」共匹配 %d 条\n", kw, list.Total)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagPage, "page", "p", 1, "页码")
	cmd.Flags().IntVar(&flagPageSize, "page-size", 20, "每页记录数（最大 50）")
	return cmd
}
