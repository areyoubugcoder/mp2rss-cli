// Package mp implements `mp2rss mp ...` subcommands.
package mp

import (
	"github.com/areyoubugcoder/mp2rss-cli/internal/client"
	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/config"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/spf13/cobra"
)

// NewCmd builds the mp command tree.
func NewCmd(deps *cliopts.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mp",
		Short: "管理订阅的公众号（list / search / subscribe / remove / articles）",
	}
	cmd.AddCommand(newListCmd(deps))
	cmd.AddCommand(newSearchCmd(deps))
	cmd.AddCommand(newSubscribeCmd(deps))
	cmd.AddCommand(newRemoveCmd(deps))
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
