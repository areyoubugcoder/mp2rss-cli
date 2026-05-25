// Package x implements `mp2rss x ...` subcommands for X (Twitter) sources.
package x

import (
	"github.com/areyoubugcoder/mp2rss-cli/internal/client"
	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/config"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/spf13/cobra"
)

// NewCmd builds the x command tree.
//
// X 账号搜索与订阅 / 取消订阅仅在 Web 控制台提供——CLI 与 Open API 都不
// 暴露这些写类端点。本命令组只暴露读类操作：list 已订阅账号、posts /
// articles 拉取已订阅账号的内容。
func NewCmd(deps *cliopts.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "x",
		Short: "管理订阅的 X 账号（list / posts / articles）",
		Long: `mp2rss x 是 X（Twitter）信息源的命令组。

仅暴露读类操作：list 已订阅账号、分页拉推文与长文。
X 账号搜索与订阅 / 取消订阅请在 Web 控制台完成（CLI 不支持）。
所有子命令复用全局 -o/--output flag（table | json）。`,
	}
	cmd.AddCommand(newListCmd(deps))
	cmd.AddCommand(newPostsCmd(deps))
	cmd.AddCommand(newArticlesCmd(deps))
	return cmd
}

// newAuthedClient builds an HTTP client using the persisted Feed Key.
// Returns CodeAuth if the user is not logged in.
func newAuthedClient() (*client.Client, error) {
	cfg := config.Get()
	key := cfg.EffectiveFeedKey()
	if key == "" {
		return nil, errs.Newf(errs.CodeAuth, "未登录。运行：mp2rss auth login")
	}
	return client.New(cfg.EffectiveAPIURL(), key), nil
}
