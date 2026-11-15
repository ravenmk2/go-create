package cli

import (
	"fmt"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
	"github.com/spf13/cobra"
)

func runPreview(cmd *cobra.Command, tmpl *template.Template, p generate.Params) error {
	plan, err := generate.Preview(tmpl, p)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Preview: %s (%s)\nDirectory: %s\n\n", p.Name, tmpl.Name, plan.Dir)
	for _, file := range plan.Files {
		action := "create"
		if file.Overwrite {
			action = "overwrite"
		}
		_, _ = fmt.Fprintf(out, "  %-9s %s\n", action, file.Path)
	}
	_, _ = fmt.Fprintf(out, "\n%d files · %d overwrites\n", len(plan.Files), len(plan.Overwrites()))
	for _, warning := range plan.Warnings {
		_, _ = fmt.Fprintf(out, "! %s\n", warning)
	}
	_, _ = fmt.Fprintln(out, "Dry run complete. No files written or commands executed.")
	return nil
}
