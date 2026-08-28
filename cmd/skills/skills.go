// Package skills implements the `mp2rss skills` command group: sync the
// repo's Agent Skills to the local agent config, show sync status, and list
// the official skills.
package skills

import (
	"fmt"

	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/areyoubugcoder/mp2rss-cli/internal/skills"
	"github.com/areyoubugcoder/mp2rss-cli/internal/version"
	"github.com/spf13/cobra"
)

// NewCmd returns the `mp2rss skills` command group.
func NewCmd(deps *cliopts.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Agent Skills 同步与查看",
		Long: `管理本仓库随 CLI 发版的 Agent Skills（mp2rss-auth / mp2rss-mp / mp2rss-x）：
CLI 自更新只替换二进制，本地已安装的 skills 不会自动跟着升级。
用 sync 同步到当前 CLI 版本，status 查看是否漂移，list 列出可用 skills。`,
	}
	cmd.AddCommand(newSyncCmd(deps))
	cmd.AddCommand(newStatusCmd(deps))
	cmd.AddCommand(newListCmd(deps))
	return cmd
}

type syncDTO struct {
	Synced   bool     `json:"synced"`
	UpToDate bool     `json:"upToDate"`
	Version  string   `json:"version"`
	Scope    string   `json:"scope"`
	Skills   []string `json:"skills,omitempty"`
}

func newSyncCmd(deps *cliopts.Deps) *cobra.Command {
	var (
		flagGlobal bool
		flagForce  bool
	)
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "把本仓库的 skills 同步 / 更新到本地 agent 配置",
		Long: `通过 npx skills add ` + skills.RepoSlug + ` 安装 / 更新本地 Agent Skills。
默认装到当前项目（./.agents/skills）；--global 装到全局（~/.claude/skills）。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope := skills.ScopeProject
			if flagGlobal {
				scope = skills.ScopeGlobal
			}
			cur := version.Version
			w := cmd.OutOrStdout()

			// 已同步且非强制：直接告知，省一次 npx 子进程。
			if st := skills.ReadState(); !flagForce && st != nil &&
				st.Scope == scope && len(st.Skills) > 0 &&
				version.Compare(st.Version, cur) == 0 {
				dto := syncDTO{Synced: false, UpToDate: true, Version: cur, Scope: string(scope)}
				if deps.Output() == output.FormatJSON {
					return output.JSON(w, dto)
				}
				fmt.Fprintf(w, "✓ 本地 skills 已是最新 %s（%s），使用 --force 强制重新同步\n", cur, scope)
				return nil
			}

			if deps.Output() != output.FormatJSON {
				fmt.Fprintf(w, "⏳ 同步 skills 到 %s（%s）…\n", cur, scope)
			}
			res, err := skills.Sync(scope, cur)
			if err != nil {
				return err
			}
			dto := syncDTO{Synced: true, Version: res.Version, Scope: string(res.Scope), Skills: res.Skills}
			if deps.Output() == output.FormatJSON {
				return output.JSON(w, dto)
			}
			fmt.Fprintf(w, "✓ 已同步 %d 个 skills 到 %s（%s）\n", len(res.Skills), res.Version, res.Scope)
			return nil
		},
	}
	cmd.Flags().BoolVar(&flagGlobal, "global", false, "安装到全局（~/.claude/skills），默认仅当前项目（./.agents/skills）")
	cmd.Flags().BoolVar(&flagForce, "force", false, "即使本地已是当前版本也重新同步")
	return cmd
}

type statusDTO struct {
	CLIVersion    string `json:"cliVersion"`
	SyncedVersion string `json:"syncedVersion,omitempty"`
	Scope         string `json:"scope,omitempty"`
	SyncedAt      string `json:"syncedAt,omitempty"`
	InSync        bool   `json:"inSync"`
}

func newStatusCmd(deps *cliopts.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看本地 skills 同步状态（是否与当前 CLI 版本一致）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			cur := version.Version
			st := skills.ReadState()
			dto := statusDTO{CLIVersion: cur}
			if st != nil {
				dto.SyncedVersion = st.Version
				dto.Scope = string(st.Scope)
				dto.SyncedAt = st.SyncedAt
				dto.InSync = version.Compare(st.Version, cur) == 0
			}
			if deps.Output() == output.FormatJSON {
				return output.JSON(w, dto)
			}
			fmt.Fprintf(w, "CLI 版本：%s\n", dto.CLIVersion)
			if st == nil {
				fmt.Fprintln(w, "本地 skills：未通过 mp2rss 同步过")
				fmt.Fprintln(w, "同步：mp2rss skills sync")
				return nil
			}
			fmt.Fprintf(w, "本地 skills 版本：%s（%s，%s）\n", dto.SyncedVersion, dto.Scope, dto.SyncedAt)
			if dto.InSync {
				fmt.Fprintln(w, "状态：✓ 已同步")
			} else {
				fmt.Fprintf(w, "状态：⚠ 落后，运行 `mp2rss skills sync` 同步到 %s\n", cur)
			}
			return nil
		},
	}
}

func newListCmd(deps *cliopts.Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出本仓库提供的 skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			if deps.Output() == output.FormatJSON {
				return output.JSON(w, skills.Official)
			}
			rows := make([][]string, len(skills.Official))
			for i, s := range skills.Official {
				rows[i] = []string{s.Name, s.Description}
			}
			output.Render(w, []output.Column{{Header: "skill", Width: 14}, {Header: "说明", Width: 60}}, rows)
			return nil
		},
	}
}
