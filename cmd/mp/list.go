package mp

import (
	"fmt"
	"strconv"

	"github.com/areyoubugcoder/mp2rss-cli/internal/client"
	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/spf13/cobra"
)

func newListCmd(deps *cliopts.Deps) *cobra.Command {
	var (
		flagQuery    string
		flagPage     int
		flagPageSize int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "查询当前订阅的公众号列表",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := newAuthedClient()
			if err != nil {
				return err
			}
			list, err := c.ListSubscriptions(flagQuery, flagPage, flagPageSize)
			if err != nil {
				return err
			}

			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), list)
			}
			renderSubscriptionTable(cmd, list.Items)
			fmt.Fprintf(cmd.OutOrStdout(), "\n共 %d 条，第 %d 页，每页 %d 条\n",
				list.Total, list.Page, list.PageSize)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagQuery, "query", "q", "", "按公众号名称模糊搜索")
	cmd.Flags().IntVarP(&flagPage, "page", "p", 1, "页码")
	cmd.Flags().IntVar(&flagPageSize, "page-size", 20, "每页记录数（最大 50）")
	return cmd
}

func renderSubscriptionTable(cmd *cobra.Command, items []client.Subscription) {
	cols := []output.Column{
		{Header: "MP_ID", Width: 10},
		{Header: "公众号", Width: 22},
		{Header: "最新文章", Width: 18},
		{Header: "订阅时间", Width: 18},
	}
	rows := make([][]string, 0, len(items))
	for _, s := range items {
		rows = append(rows, []string{
			strconv.FormatInt(s.MpID, 10),
			s.MpName,
			output.FormatUnixMillis(s.MpLastArticleAt),
			output.FormatUnixMillis(s.CreatedAt),
		})
	}
	output.Render(cmd.OutOrStdout(), cols, rows)
}
