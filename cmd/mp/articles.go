package mp

import (
	"fmt"
	"strconv"

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
		Use:   "articles <mpId>",
		Short: "查询指定公众号的文章列表",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mpID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil || mpID <= 0 {
				return errs.Newf(errs.CodeArgs, "mpId 必须为正整数：%q", args[0])
			}
			c, err := newAuthedClient()
			if err != nil {
				return err
			}
			list, err := c.ListArticles(mpID, flagPage, flagPageSize)
			if err != nil {
				return err
			}
			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), list)
			}

			cols := []output.Column{
				{Header: "发布时间", Width: 18},
				{Header: "标题", Width: 40},
				{Header: "链接", Width: 50},
			}
			rows := make([][]string, 0, len(list.Items))
			for _, a := range list.Items {
				rows = append(rows, []string{
					output.FormatUnixMillis(a.PublishedAt),
					a.Title,
					a.OriginalURL,
				})
			}
			output.Render(cmd.OutOrStdout(), cols, rows)
			fmt.Fprintf(cmd.OutOrStdout(), "\n共 %d 篇\n", len(list.Items))
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagPage, "page", "p", 1, "页码")
	cmd.Flags().IntVar(&flagPageSize, "page-size", 100, "每页记录数（最大 100）")
	return cmd
}
