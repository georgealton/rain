package rain

import (
	"fmt"

	"github.com/georgealton/rain/internal/config"
	"github.com/spf13/cobra"

	"github.com/georgealton/rain/internal/aws/s3"
	"github.com/georgealton/rain/internal/cmd"
	"github.com/georgealton/rain/internal/cmd/bootstrap"
	"github.com/georgealton/rain/internal/cmd/build"
	"github.com/georgealton/rain/internal/cmd/cat"
	"github.com/georgealton/rain/internal/cmd/cc"
	consolecmd "github.com/georgealton/rain/internal/cmd/console"
	"github.com/georgealton/rain/internal/cmd/deploy"
	"github.com/georgealton/rain/internal/cmd/diff"
	rainfmt "github.com/georgealton/rain/internal/cmd/fmt"
	"github.com/georgealton/rain/internal/cmd/forecast"
	"github.com/georgealton/rain/internal/cmd/info"
	"github.com/georgealton/rain/internal/cmd/logs"
	"github.com/georgealton/rain/internal/cmd/ls"
	"github.com/georgealton/rain/internal/cmd/merge"
	"github.com/georgealton/rain/internal/cmd/module"
	"github.com/georgealton/rain/internal/cmd/pkg"
	"github.com/georgealton/rain/internal/cmd/rm"
	"github.com/georgealton/rain/internal/cmd/stackset"
	"github.com/georgealton/rain/internal/cmd/tree"
	"github.com/georgealton/rain/internal/cmd/watch"
	"github.com/georgealton/rain/internal/console"
)

// Cmd is the rain command's entrypoint
var Cmd = &cobra.Command{
	Use:     "rain",
	Long:    "Rain is a command line tool for working with AWS CloudFormation templates and stacks",
	Version: config.VERSION,
}

const usageTemplate = `Usage:{{if .Runnable}}
  <cyan>{{.UseLine}}</>{{end}}{{if .HasAvailableSubCommands}}
  <cyan>{{.CommandPath}}</> [<gray>command</>]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{range $group := groups}}{{ $group }}:{{range $c := $.Commands}}{{if $c.IsAvailableCommand}}{{if eq $c.Annotations.Group $group}}
  <cyan>{{rpad $c.Name $c.NamePadding }}</> {{$c.Short}}{{end}}{{end}}{{end}}

{{end}}Other Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}{{if .Annotations.Group}}{{else}}
  <cyan>{{rpad .Name .NamePadding }}</> {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if and .HasParent .HasAvailableFlags}}

Flags:
{{.Flags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}
`

const stackGroup = "Stack commands"
const templateGroup = "Template commands"

func addCommand(label string, profileOptions, bucketOptions bool, c *cobra.Command) {
	if label != "" {
		c.Annotations = map[string]string{"Group": label}
	}

	if profileOptions {
		c.Flags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
		c.Flags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")
	}

	if bucketOptions {
		c.Flags().StringVar(&s3.BucketName, "s3-bucket", "", "Name of the S3 bucket that is used to upload assets")
		c.Flags().StringVar(&s3.BucketKeyPrefix, "s3-prefix", "", "Prefix to add to objects uploaded to S3 bucket")
		c.Flags().StringVar(&s3.ExpectedBucketOwner, "s3-owner", "", "The account where S3 assets are stored")
	}

	Cmd.AddCommand(c)
}

func init() {
	// Stack commands
	addCommand(stackGroup, true, false, cat.Cmd)
	addCommand(stackGroup, true, true, deploy.Cmd)
	addCommand(stackGroup, true, true, cc.Cmd)
	addCommand(stackGroup, true, false, logs.Cmd)
	addCommand(stackGroup, true, false, ls.Cmd)
	addCommand(stackGroup, true, false, rm.Cmd)
	addCommand(stackGroup, true, false, watch.Cmd)
	addCommand(stackGroup, true, false, stackset.StackSetCmd)

	// Template commands
	addCommand(templateGroup, true, true, bootstrap.Cmd)
	addCommand(templateGroup, true, false, build.Cmd)
	addCommand(templateGroup, false, false, diff.Cmd)
	addCommand(templateGroup, false, false, rainfmt.Cmd)
	addCommand(templateGroup, false, false, merge.Cmd)
	addCommand(templateGroup, true, true, pkg.Cmd)
	addCommand(templateGroup, false, false, tree.Cmd)
	addCommand(templateGroup, true, false, forecast.Cmd)
	addCommand(templateGroup, true, false, module.Cmd)

	// Other commands
	addCommand("", true, false, consolecmd.Cmd)
	addCommand("", true, false, info.Cmd)

	// Customise usage
	Cmd.Annotations = map[string]string{"Groups": fmt.Sprintf("%s|%s", stackGroup, templateGroup)}

	cobra.AddTemplateFunc("groups", func() []string {
		return []string{stackGroup, templateGroup}
	})

	oldUsageFunc := Cmd.UsageFunc()
	Cmd.SetUsageFunc(func(c *cobra.Command) error {
		Cmd.SetUsageTemplate(console.Sprint(usageTemplate))
		return oldUsageFunc(c)
	})

	Cmd.PersistentFlags().BoolVarP(&console.NoColour, "no-colour", "", false, "Disable colour output")

	cmd.AddDefaults(Cmd)
}
