// Package cmd wires the cobra command tree for mp2rss CLI.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/areyoubugcoder/mp2rss-cli/cmd/auth"
	"github.com/areyoubugcoder/mp2rss-cli/cmd/mp"
	"github.com/areyoubugcoder/mp2rss-cli/cmd/update"
	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/config"
	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/areyoubugcoder/mp2rss-cli/internal/version"
	"github.com/spf13/cobra"
)

// Persistent flag values.
var (
	flagOutput string
	flagAPIKey string
	flagAPIURL string
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mp2rss",
		Short: "mp2rss CLI — 微信公众号 RSS 订阅管理",
		Long: `mp2rss CLI 是 mp2rss.bugcode.dev 的命令行客户端。
登录后可订阅公众号、查询文章、管理订阅列表，输出支持表格与 JSON 两种格式。

支持环境变量：
  MP2RSS_FEED_KEY   覆盖 Feed Key（优先级高于配置文件）
  MP2RSS_API_URL    覆盖 API 地址`,
		Version:           version.String(),
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: persistentPreRunE,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}

	root.PersistentFlags().StringVarP(&flagOutput, "output", "o", output.FormatTable, "输出格式：table 或 json")
	root.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "覆盖 Feed Key（也可使用 MP2RSS_FEED_KEY 环境变量）")
	root.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "覆盖 API 地址（默认 https://mp2rss.bugcode.dev）")

	deps := &cliopts.Deps{
		Output: func() string { return flagOutput },
		APIKey: func() string { return flagAPIKey },
		APIURL: func() string { return flagAPIURL },
	}

	root.AddCommand(auth.NewCmd(deps))
	root.AddCommand(mp.NewCmd(deps))
	root.AddCommand(update.NewCmd())

	return root
}

// Execute runs the root command and exits with the mapped code on error.
func Execute() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		code := errs.CodeGeneric
		msg := err.Error()
		var typed *errs.Error
		if errors.As(err, &typed) {
			code = typed.Code
		}
		// JSON mode: error envelope on stdout for jq compatibility.
		if flagOutput == output.FormatJSON {
			httpOrCode := code
			if typed != nil && typed.HTTPStatus != 0 {
				httpOrCode = typed.HTTPStatus
			}
			output.PrintErrorJSON(os.Stdout, msg, httpOrCode)
		} else {
			fmt.Fprintln(os.Stderr, "✗", msg)
		}
		os.Exit(code)
	}
}

func persistentPreRunE(_ *cobra.Command, _ []string) error {
	if err := output.Validate(flagOutput); err != nil {
		return errs.Wrap(errs.CodeArgs, err)
	}
	cfg := config.Get()
	if flagAPIKey != "" {
		cfg.FeedKey = flagAPIKey
	}
	if flagAPIURL != "" {
		cfg.APIURL = flagAPIURL
	}
	return nil
}
