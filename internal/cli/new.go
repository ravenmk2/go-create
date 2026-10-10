package cli

import (
	"github.com/spf13/cobra"

	"github.com/ravenmk2/go-create/internal/generate"
	"github.com/ravenmk2/go-create/internal/template"
	"github.com/ravenmk2/go-create/internal/wizard"
)

func newCmd() *cobra.Command {
	var (
		module, dir, license, holder string
		sets                         []string
		yes, noGit, noTidy           bool
	)
	c := &cobra.Command{
		Use:   "new <template> [project-name]",
		Short: "Create a new project (missing parameters are filled in by the wizard; --yes generates directly with complete parameters)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpl, err := template.Load(args[0])
			if err != nil {
				return err
			}
			p := generate.Params{
				Template: tmpl.Name,
				Module:   module,
				Dir:      dir,
				License:  license,
				Holder:   holder,
				NoGit:    noGit,
				NoTidy:   noTidy,
			}
			if len(args) == 2 {
				p.Name = args[1]
			}
			raw, err := parseSetFlags(sets)
			if err != nil {
				return err
			}
			// Validate --set as a whole first: unknown keys / invalid values fail fast
			if _, err := tmpl.CoerceOptions(raw); err != nil {
				return err
			}
			p.Options = raw

			if yes {
				if err := generate.Validate(tmpl, &p); err != nil {
					return err
				}
			} else {
				templates, err := template.LoadAll()
				if err != nil {
					return err
				}
				if tmpl, err = wizard.Run(templates, &p); err != nil {
					return err
				}
				if err := confirmNonEmptyDir(&p); err != nil {
					return err
				}
			}
			return runGenerate(cmd, tmpl, p)
		},
	}
	f := c.Flags()
	f.StringVar(&module, "module", "", "module path (e.g. github.com/user/project)")
	f.StringVar(&dir, "dir", "", "target directory (default ./<project-name>; '.' means the current directory)")
	f.StringVar(&license, "license", "", "license: mit / apache-2.0 / gpl-3.0 / none")
	f.StringVar(&holder, "holder", "", "license copyright holder (required when a license is selected)")
	f.StringArrayVar(&sets, "set", nil, "template options, Helm style: key=value[,k2=v2], repeatable; use {a,b} for lists")
	f.BoolVar(&yes, "yes", false, "skip the wizard and confirmation and generate directly (fails if parameters are incomplete)")
	f.BoolVar(&noGit, "no-git", false, "skip git init")
	f.BoolVar(&noTidy, "no-tidy", false, "skip go mod tidy")
	return c
}
