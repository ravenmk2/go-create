package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/ravenmk2/go-create/internal/upgrade"
)

// upgradeTheme holds the lipgloss styles for the upgrade command's TTY
// output. With color off every style is the identity.
type upgradeTheme struct {
	version lipgloss.Style // blue: version numbers
	path    lipgloss.Style // blue: asset names and file paths
	success lipgloss.Style // green: success lines
}

func newUpgradeTheme(color bool) upgradeTheme {
	r := lipgloss.DefaultRenderer()
	if color {
		// The color decision is already made via isatty; force the profile
		// so lipgloss's own detection cannot veto it.
		r.SetColorProfile(termenv.TrueColor)
	}
	style := func(hex string) lipgloss.Style {
		s := r.NewStyle()
		if color {
			return s.Foreground(lipgloss.Color(hex))
		}
		return s
	}
	return upgradeTheme{
		version: style("#2D9CDB"),
		path:    style("#2D9CDB"),
		success: style("#04B575"),
	}
}

// upgradeCmd implements `go-create upgrade`: self-update from GitHub
// releases. Confirmation is a huh prompt on a TTY unless --yes; off a TTY
// without --yes the command fails instead of waiting for input.
func upgradeCmd(version string) *cobra.Command {
	var (
		check, yes bool
		pinVersion string
		attempts   int
		timeout    time.Duration
	)
	c := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade go-create to the latest or a specified release",
		Args:  cobra.NoArgs,
		Example: `  go-create upgrade --check                 # report availability only
  go-create upgrade --yes                   # upgrade to the latest release
  go-create upgrade --version v1.2.3 --yes  # pin a target version`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if attempts < 1 {
				return errors.New("invalid --attempts value: must be at least 1")
			}

			stdoutTTY := isatty.IsTerminal(os.Stdout.Fd())
			theme := newUpgradeTheme(stdoutTTY)
			out := cmd.OutOrStdout()

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			client := upgrade.NewClient()
			opts := upgrade.Options{
				Current:  version,
				Version:  pinVersion,
				Attempts: attempts,
			}
			plan, err := upgrade.Resolve(ctx, client, opts)
			if err != nil {
				return err
			}

			if plan.UpToDate {
				msg := fmt.Sprintf("go-create is already up to date (%s)", plan.From)
				if plan.Newer {
					msg = fmt.Sprintf("current version (%s) is newer than the latest release (%s)", plan.From, plan.To)
				}
				_, _ = fmt.Fprintf(out, "%s\n", theme.success.Render("✓ "+msg))
				return nil
			}
			if check {
				_, _ = fmt.Fprintf(out, "go-create %s is available (current: %s)\nrun %s to upgrade\n",
					theme.version.Render(plan.To), theme.version.Render(plan.From),
					"go-create upgrade")
				return nil
			}

			target, err := upgrade.TargetPath()
			if err != nil {
				return err
			}
			opts.Target = target
			_, _ = fmt.Fprintf(out, "%-10s%s\n", "upgrade:", theme.version.Render(plan.From+" → "+plan.To))
			_, _ = fmt.Fprintf(out, "%-10s%s\n", "package:", theme.path.Render(plan.Asset))
			_, _ = fmt.Fprintf(out, "%-10s%s\n", "location:", theme.path.Render(target))

			if !yes {
				if !isatty.IsTerminal(os.Stdin.Fd()) {
					return errors.New("upgrade requires confirmation but stdin is not interactive; re-run with --yes to confirm the upgrade")
				}
				confirm := true
				form := huh.NewForm(huh.NewGroup(
					huh.NewConfirm().
						Title(fmt.Sprintf("Upgrade go-create from %s to %s?", plan.From, plan.To)).
						Value(&confirm),
				))
				if err := form.Run(); err != nil {
					return err
				}
				if !confirm {
					_, _ = fmt.Fprintln(out, "upgrade canceled")
					return nil
				}
			}

			bar := upgrade.NewBar(os.Stderr, isatty.IsTerminal(os.Stderr.Fd()), stdoutTTY)
			res, err := upgrade.Apply(ctx, client, plan, opts, bar.Update)
			bar.Finish()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(out, "%s\n", theme.success.Render("✓ upgrade complete"))
			if res.Backup != "" {
				_, _ = fmt.Fprintf(out, "%-10s%s\n", "backup:", theme.path.Render(res.Backup))
			}
			return nil
		},
	}
	f := c.Flags()
	f.BoolVar(&check, "check", false, "check for updates without upgrading")
	f.BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	f.StringVar(&pinVersion, "version", "", "target version (default: latest release)")
	f.IntVar(&attempts, "attempts", 10, "download attempts before giving up")
	f.DurationVar(&timeout, "timeout", 5*time.Minute, "overall timeout for the upgrade")
	return c
}
