package x

import (
	"fmt"

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
		Short: "查询当前已订阅的 X 账号列表",
		Long: `列出当前 Feed Key 下已订阅的全部 X 账号。

底层调 GET /open-api/subscriptions?sourceType=x（server 端按源过滤）。
若 server 不识别 sourceType 参数，会回退为客户端筛选（剔除 mp 项）。`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := newAuthedClient()
			if err != nil {
				return err
			}
			list, err := c.ListSubscriptionsFiltered(flagQuery, "x", flagPage, flagPageSize)
			if err != nil {
				return err
			}

			// 客户端兜底过滤：极端情况下 server 没认 sourceType=x，把 mp 项也带回来。
			xItems := filterXOnly(list.Items)
			list.Items = xItems

			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), list)
			}
			renderXSubscriptionTable(cmd, xItems)
			fmt.Fprintf(cmd.OutOrStdout(), "\n共 %d 条，第 %d 页，每页 %d 条\n",
				len(xItems), list.Page, list.PageSize)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagQuery, "query", "q", "", "按 displayName / username 模糊搜索（仅服务端支持时生效）")
	cmd.Flags().IntVarP(&flagPage, "page", "p", 1, "页码")
	cmd.Flags().IntVar(&flagPageSize, "page-size", 20, "每页记录数（最大 50）")
	return cmd
}

func filterXOnly(items []client.Subscription) []client.Subscription {
	out := make([]client.Subscription, 0, len(items))
	for _, s := range items {
		// server 升级前可能不返回 sourceType；用 XUserID 非空作为兜底判定。
		if s.SourceType == "x" || s.XUserID != "" {
			out = append(out, s)
		}
	}
	return out
}

func renderXSubscriptionTable(cmd *cobra.Command, items []client.Subscription) {
	cols := []output.Column{
		{Header: "X_USER_ID", Width: 16},
		{Header: "用户名", Width: 18},
		{Header: "显示名", Width: 22},
		{Header: "认证", Width: 4},
		{Header: "最新内容", Width: 18},
		{Header: "订阅时间", Width: 18},
	}
	rows := make([][]string, 0, len(items))
	for _, s := range items {
		verified := "-"
		if s.XVerified {
			verified = "✓"
		}
		username := s.XUsername
		if username != "" {
			username = "@" + username
		}
		rows = append(rows, []string{
			s.XUserID,
			username,
			s.XDisplayName,
			verified,
			output.FormatUnixMillis(s.XLastItemAt),
			output.FormatUnixMillis(s.CreatedAt),
		})
	}
	output.Render(cmd.OutOrStdout(), cols, rows)
}
