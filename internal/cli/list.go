package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravenmk2/go-create/internal/template"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available templates and their options",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			templates, err := template.LoadAll()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, t := range templates {
				_, _ = fmt.Fprintf(out, "%s [%s]\n  %s\n", t.Name, t.Source, t.Description)
				for _, o := range t.Options {
					_, _ = fmt.Fprintf(out, "    --set %-16s %-11s default %s  %s\n",
						o.Key, string(o.Type), defaultDisplay(o), o.Prompt)
				}
			}
			if dir, err := template.UserTemplateDir(); err == nil {
				_, _ = fmt.Fprintf(out, "\nCustom template directory: %s\n", dir)
			}
			return nil
		},
	}
}

func defaultDisplay(o template.Option) string {
	if o.Default == nil {
		return "(none)"
	}
	switch v := o.Default.(type) {
	case []any:
		strs := make([]string, 0, len(v))
		for _, e := range v {
			strs = append(strs, fmt.Sprint(e))
		}
		return "{" + strings.Join(strs, ",") + "}"
	default:
		return fmt.Sprint(v)
	}
}
