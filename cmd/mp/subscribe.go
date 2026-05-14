package mp

import (
	"fmt"

	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/spf13/cobra"
)

type subscribeResult struct {
	OK         bool   `json:"ok"`
	ArticleURL string `json:"article_url"`
}

func newSubscribeCmd(deps *cliopts.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "subscribe <article-url>",
		Short: "通过一篇公众号文章 URL 订阅其所属公众号",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			if url == "" {
				return errs.Newf(errs.CodeArgs, "URL 不能为空")
			}
			c, err := newAuthedClient()
			if err != nil {
				return err
			}
			if err := c.Subscribe(url); err != nil {
				return err
			}
			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), subscribeResult{OK: true, ArticleURL: url})
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✓ 订阅成功")
			return nil
		},
	}
}
