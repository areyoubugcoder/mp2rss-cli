package mp

import (
	"fmt"
	"strconv"

	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/areyoubugcoder/mp2rss-cli/internal/prompt"
	"github.com/spf13/cobra"
)

type removeResult struct {
	OK   bool  `json:"ok"`
	MpID int64 `json:"mpId"`
}

func newRemoveCmd(deps *cliopts.Deps) *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "remove <mpId>",
		Short: "取消订阅指定 mpId",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mpID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil || mpID <= 0 {
				return errs.Newf(errs.CodeArgs, "mpId 必须为正整数：%q", args[0])
			}

			if !flagYes && deps.Output() == output.FormatTable {
				ok, err := prompt.Confirm(fmt.Sprintf("确认取消订阅 mpId=%d？", mpID), false, cmd.OutOrStdout())
				if err != nil {
					return errs.Wrap(errs.CodeGeneric, err)
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "已取消。")
					return nil
				}
			}

			c, err := newAuthedClient()
			if err != nil {
				return err
			}
			if err := c.Unsubscribe(mpID); err != nil {
				return err
			}
			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), removeResult{OK: true, MpID: mpID})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ 已取消订阅 mpId=%d\n", mpID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "跳过交互确认")
	return cmd
}
