package x

import (
	"fmt"

	"github.com/areyoubugcoder/mp2rss-cli/internal/client"
	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPostsCmd(deps *cliopts.Deps) *cobra.Command {
	var (
		flagPage     int
		flagPageSize int
	)

	cmd := &cobra.Command{
		Use:   "posts <xUserId>",
		Short: "分页查询已订阅 X 账号的推文流",
		Long: `按 postedAt DESC 拉取已订阅 X 账号的推文列表。

未订阅的 xUserId 会返回 404 "X account is not subscribed"。
返回的是结构化原始数据（含 media / quotedPost / threadPosts），
适合脚本消费；要订阅后供阅读器订阅请走 Feed 公开层。`,
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
			list, err := c.XListPosts(id, flagPage, flagPageSize)
			if err != nil {
				return err
			}
			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), list)
			}
			renderPostsTable(cmd, list.Items)
			fmt.Fprintf(cmd.OutOrStdout(), "\n共 %d 条，第 %d 页，每页 %d 条\n",
				list.Total, list.Page, list.PageSize)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagPage, "page", "p", 1, "页码")
	cmd.Flags().IntVar(&flagPageSize, "page-size", 20, "每页记录数（最大 50）")
	return cmd
}

func renderPostsTable(cmd *cobra.Command, items []client.XPost) {
	cols := []output.Column{
		{Header: "发布时间", Width: 18},
		{Header: "推文 ID", Width: 22},
		{Header: "内容", Width: 50},
		{Header: "线程", Width: 4},
	}
	rows := make([][]string, 0, len(items))
	for _, p := range items {
		thread := "-"
		if len(p.ThreadPosts) > 0 {
			thread = fmt.Sprintf("%d", len(p.ThreadPosts))
		}
		rows = append(rows, []string{
			output.FormatUnixMillis(p.PostedAt),
			p.PostID,
			p.Content,
			thread,
		})
	}
	output.Render(cmd.OutOrStdout(), cols, rows)
}
