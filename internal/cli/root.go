// Package cli implements the command-line entry point of go-create.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
	"github.com/ravenmk2/go-create/internal/wizard"
)

// Execute is the command-line entry point; version is injected at link time.
func Execute(version string) {
	root := &cobra.Command{
		Use:          "go-create",
		Short:        "Bootstrap Go projects per engineering conventions",
		Version:      version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := generate.Params{}
			tmpl, err := runWizard(&p)
			if err != nil {
				return err
			}
			if err := confirmNonEmptyDir(&p); err != nil {
				return err
			}
			return runGenerate(cmd, tmpl, p)
		},
	}
	root.AddCommand(newCmd(), listCmd(), upgradeCmd(version))
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// runWizard loads templates and runs the wizard to fill in missing parameters.
func runWizard(p *generate.Params) (*template.Template, error) {
	templates, err := template.LoadAll()
	if err != nil {
		return nil, err
	}
	return wizard.Run(templates, p)
}

// runGenerate runs generation and prints the result.
func runGenerate(cmd *cobra.Command, tmpl *template.Template, p generate.Params) error {
	res, err := generate.Generate(tmpl, p)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "\nCreated project %s (%d files, directory %s)\n", p.Name, len(res.Files), res.Dir)
	for _, w := range res.Warnings {
		_, _ = fmt.Fprintf(out, "warning: %s\n", w)
	}
	return nil
}
