package auth

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/areyoubugcoder/mp2rss-cli/internal/authflow"
	"github.com/areyoubugcoder/mp2rss-cli/internal/browser"
	"github.com/areyoubugcoder/mp2rss-cli/internal/client"
	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/config"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/prompt"
	"github.com/areyoubugcoder/mp2rss-cli/internal/version"
	"github.com/spf13/cobra"
)

// defaultWebOrigin hosts the /cli/authorize page. Overridable via --web-origin
// (hidden flag) for local development.
const defaultWebOrigin = "https://mp2rss.bugcode.dev"

func newLoginCmd(deps *cliopts.Deps) *cobra.Command {
	_ = deps
	var (
		flagFeedKey  string
		flagNoBrowse bool
		flagWeb      string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "登录 mp2rss（默认走浏览器 OAuth Loopback 流）",
		Long: `登录方式：

  mp2rss auth login                       # 默认：浏览器授权
  mp2rss auth login -k <feed-key>         # 直接落盘，跳过浏览器
  mp2rss auth login --no-browser          # 远程/SSH：仅打印 URL，手动粘贴 Feed Key`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			cfg := config.Get()

			if flagFeedKey != "" {
				return saveAndVerify(out, cfg, flagFeedKey)
			}
			if flagNoBrowse {
				return runManualPaste(out, cfg, flagWeb)
			}
			return runLoopback(cmd.Context(), out, cfg, flagWeb)
		},
	}

	cmd.Flags().StringVarP(&flagFeedKey, "feed-key", "k", "", "直接保存 Feed Key 跳过浏览器（CI / 无头环境）")
	cmd.Flags().BoolVar(&flagNoBrowse, "no-browser", false, "不打开浏览器，仅打印授权 URL（SSH 远程使用）")
	cmd.Flags().StringVar(&flagWeb, "web-origin", defaultWebOrigin, "Web 端授权页面来源（用于本地联调）")
	_ = cmd.Flags().MarkHidden("web-origin")
	return cmd
}

func runLoopback(ctx context.Context, out io.Writer, cfg *config.Config, webOrigin string) error {
	flow, err := authflow.New(webOrigin)
	if err != nil {
		return errs.Wrap(errs.CodeGeneric, err)
	}
	defer flow.Close()

	authURL := flow.AuthorizeURL(version.String())
	fmt.Fprintln(out, "🔗 正在等待浏览器授权 mp2rss CLI…")
	fmt.Fprintf(out, "   授权 URL：%s\n", authURL)
	fmt.Fprintln(out, "   若浏览器未自动打开，请手动复制此 URL。")
	_ = browser.Open(authURL)

	if ctx == nil {
		ctx = context.Background()
	}
	res, err := flow.Wait(ctx)
	if err != nil {
		return errs.Newf(errs.CodeGeneric, "%s", err.Error())
	}
	return saveAndVerify(out, cfg, res.FeedKey)
}

func runManualPaste(out io.Writer, cfg *config.Config, webOrigin string) error {
	fmt.Fprintln(out, "🔗 在浏览器打开下方链接，登录后从「设置 → Feed Key」复制密钥粘贴回此处。")
	fmt.Fprintf(out, "   %s/settings\n", webOrigin)
	key, err := prompt.Secret("Feed Key（输入时不会回显）: ", out)
	if err != nil {
		return errs.Wrap(errs.CodeGeneric, err)
	}
	if key == "" {
		return errs.Newf(errs.CodeArgs, "未提供 Feed Key")
	}
	return saveAndVerify(out, cfg, key)
}

// saveAndVerify validates the key against the API, then persists on success.
// The key is never printed in plaintext.
func saveAndVerify(out io.Writer, cfg *config.Config, feedKey string) error {
	apiURL := cfg.EffectiveAPIURL()
	c := client.New(apiURL, feedKey)
	if err := c.VerifyAuth(); err != nil {
		return err
	}
	cfg.FeedKey = feedKey
	cfg.LastLoginAt = time.Now().Unix()
	cfg.LastVerifyAt = cfg.LastLoginAt
	if err := cfg.Save(); err != nil {
		return errs.Wrap(errs.CodeGeneric, err)
	}
	fmt.Fprintf(out, "✓ 已登录。Feed Key %s（已写入 ~/.mp2rss/config.json，权限 0600）\n",
		config.MaskKey(feedKey))
	fmt.Fprintf(out, "  API：%s\n", apiURL)
	return nil
}
