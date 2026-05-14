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
type statusDTO struct {
	LoggedIn        bool   `json:"logged_in"`
	Source          string `json:"source"`             // "env" | "config" | "none"
	APIURL          string `json:"api_url"`
	FeedKeyMasked   string `json:"feed_key_masked,omitempty"`
	LastVerifyAtISO string `json:"last_verify_at,omitempty"`
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
			}
			switch {
			case os.Getenv("MP2RSS_FEED_KEY") != "":
				dto.Source = "env"
			case cfg.FeedKey != "":
				dto.Source = "config"
			default:
				dto.Source = "none"
			}
			if cfg.LastVerifyAt > 0 {
				dto.LastVerifyAtISO = time.Unix(cfg.LastVerifyAt, 0).Local().Format(output.TimeFormat)
			}

			if deps.Output() == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), dto)
			}

			w := cmd.OutOrStdout()
			if !dto.LoggedIn {
				fmt.Fprintln(w, "状态：未登录")
				fmt.Fprintf(w, "API：%s\n", dto.APIURL)
				fmt.Fprintln(w, "登录：mp2rss auth login")
				return nil
			}
			fmt.Fprintf(w, "状态：已登录（来源：%s）\n", dto.Source)
			fmt.Fprintf(w, "API：%s\n", dto.APIURL)
			fmt.Fprintf(w, "Feed Key：%s\n", dto.FeedKeyMasked)
			if dto.LastVerifyAtISO != "" {
				fmt.Fprintf(w, "上次校验：%s\n", dto.LastVerifyAtISO)
			}
			return nil
		},
	}
}
