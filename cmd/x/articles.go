package x

import (
	"fmt"

	"github.com/areyoubugcoder/mp2rss-cli/internal/client"
	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/spf13/cobra"
)

func newArticlesCmd(deps *cliopts.Deps) *cobra.Command {
	var (
		flagPage     int
		flagPageSize int
	)

	cmd := &cobra.Command{
		Use:   "articles <xUserId>",
		Short: "分页查询已订阅 X 账号的长文流",
		Long: `按 publishedAt DESC 拉取已订阅 X 账号的长文（Articles）列表。

未订阅的 xUserId 会返回 404 "X account is not subscribed"。
contentMarkdown 为上游推送的 markdown 原文，可能为 null。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if id == "" {
				return errs.Newf(errs.CodeArgs, "xUserId 不能为空")
			}
			if flagPageSize > 50 {
				return errs.Newf(errs.CodeArgs, "--page-size 上限 50")
			}
			c, err := newAuthedClient()
			if err != nil {
				return err
			}
			list, err := c.XListArticles(id, flagPage, flagPageSize)
			if err != nil {
				return err
			}
			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), list)
			}
			renderArticlesTable(cmd, list.Items)
			fmt.Fprintf(cmd.OutOrStdout(), "\n共 %d 条，第 %d 页，每页 %d 条\n",
				list.Total, list.Page, list.PageSize)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagPage, "page", "p", 1, "页码")
	cmd.Flags().IntVar(&flagPageSize, "page-size", 20, "每页记录数（最大 50）")
	return cmd
}

func renderArticlesTable(cmd *cobra.Command, items []client.XArticle) {
	cols := []output.Column{
		{Header: "发布时间", Width: 18},
		{Header: "标题", Width: 36},
		{Header: "URL", Width: 50},
	}
	rows := make([][]string, 0, len(items))
	for _, a := range items {
		rows = append(rows, []string{
			output.FormatUnixMillis(a.PublishedAt),
			a.Title,
			a.URL,
		})
	}
	output.Render(cmd.OutOrStdout(), cols, rows)
}
