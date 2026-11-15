package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"

	"github.com/ravenmk2/go-create/internal/template"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available templates (use show for options)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			templates, err := template.LoadAll()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			table := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(table, "TEMPLATE\tSOURCE\tDESCRIPTION")
			for _, t := range templates {
				description := ansi.Truncate(strings.Join(strings.Fields(t.Description), " "), 52, "…")
				_, _ = fmt.Fprintf(table, "%s\t%s\t%s\n", t.Name, t.Source, description)
			}
			_ = table.Flush()
			_, _ = fmt.Fprintln(out, "\nUse go-create show <template> for descriptions, choices and examples.")
			if dir, err := template.UserTemplateDir(); err == nil {
				_, _ = fmt.Fprintf(out, "\nCustom template directory: %s\n", dir)
			}
			return nil
		},
	}
}

func showCmd() *cobra.Command {
	return &cobra.Command{
		Use: "show <template>", Short: "Show a template's options, choices and defaults", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := template.Load(args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "%s [%s]\n%s\n", t.Name, t.Source, t.Description)
			for _, o := range t.Options {
				_, _ = fmt.Fprintf(out, "\n%s (%s)\n  %s\n  Default: %s\n", o.Key, o.Type, o.Prompt, defaultDisplay(o))
				if o.Help != "" {
					_, _ = fmt.Fprintf(out, "  %s\n", o.Help)
				}
				if o.Pattern != "" {
					_, _ = fmt.Fprintf(out, "  Pattern: %s (non-empty values)\n", o.Pattern)
				}
				for _, choice := range o.Choices {
					_, _ = fmt.Fprintf(out, "  %-16s %s\n", choice.Value, choice.Label)
				}
				value := defaultDisplay(o)
				if o.Default == nil {
					value = "value"
					if o.Type == template.OptionBoolean {
						value = "true"
					}
				}
				if len(o.Choices) > 0 {
					value = o.Choices[0].Value
					if o.Type == template.OptionMultiChoice {
						value = "{" + value + "}"
					}
				}
				_, _ = fmt.Fprintf(out, "  Example: --set %s\n", shellDirectory(o.Key+"="+value))
			}
			_, _ = fmt.Fprintf(out, "\nCreate: go-create new %s myapp --module github.com/user/myapp --yes\n", t.Name)
			return nil
		},
	}
}

func defaultDisplay(o template.Option) string {
	if o.Default == nil {
		switch o.Type {
		case template.OptionBoolean:
			return "false"
		case template.OptionMultiChoice:
			return "{}"
		default:
			return "(empty)"
		}
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
