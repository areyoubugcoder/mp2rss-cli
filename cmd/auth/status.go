package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/config"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/spf13/cobra"
)

// statusDTO is the JSON shape for `auth status`.
//
// Field names are camelCase to align with the Open API DTOs; timestamps are
// emitted as unix-millis numbers (0 when absent) so jq users can `from_unixtime`
// without first reparsing a locale-formatted string.
type statusDTO struct {
	LoggedIn      bool   `json:"loggedIn"`
	Source        string `json:"source"` // "env" | "config" | "none"
	APIURL        string `json:"apiUrl"`
	FeedKeyMasked string `json:"feedKeyMasked,omitempty"`
	Name          string `json:"name,omitempty"`
	Email         string `json:"email,omitempty"`
	LastLoginAt   int64  `json:"lastLoginAt,omitempty"`  // unix millis
	LastVerifyAt  int64  `json:"lastVerifyAt,omitempty"` // unix millis
}

func newStatusCmd(deps *cliopts.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看登录状态、当前 API 地址与 Feed Key 掩码",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := config.Get()
			key := cfg.EffectiveFeedKey()

			dto := statusDTO{
				LoggedIn:      key != "",
				APIURL:        cfg.EffectiveAPIURL(),
				FeedKeyMasked: config.MaskKey(key),
				Name:          cfg.Name,
				Email:         cfg.Email,
			}
			switch {
			case os.Getenv("MP2RSS_FEED_KEY") != "":
				dto.Source = "env"
			case cfg.FeedKey != "":
				dto.Source = "config"
			default:
				dto.Source = "none"
			}
			// config 里存的是 unix 秒，公开 JSON 一律转毫秒以对齐 API DTO。
			if cfg.LastLoginAt > 0 {
				dto.LastLoginAt = cfg.LastLoginAt * 1000
			}
			if cfg.LastVerifyAt > 0 {
				dto.LastVerifyAt = cfg.LastVerifyAt * 1000
			}

			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), dto)
			}

			w := cmd.OutOrStdout()
			if !dto.LoggedIn {
				fmt.Fprintln(w, "状态：未登录")
				fmt.Fprintln(w, "登录：mp2rss auth login")
				return nil
			}
			fmt.Fprintf(w, "状态：已登录（来源：%s）\n", dto.Source)
			if dto.Name != "" {
				fmt.Fprintf(w, "用户：%s\n", dto.Name)
			}
			if dto.Email != "" {
				fmt.Fprintf(w, "邮箱：%s\n", dto.Email)
			}
			fmt.Fprintf(w, "Feed Key：%s\n", dto.FeedKeyMasked)
			if cfg.LastVerifyAt > 0 {
				fmt.Fprintf(w, "上次校验：%s\n",
					time.Unix(cfg.LastVerifyAt, 0).Local().Format(output.TimeFormat))
			}
			return nil
		},
	}
}
