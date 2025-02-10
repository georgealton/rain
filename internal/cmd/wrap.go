package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/georgealton/rain/internal/config"
	"github.com/georgealton/rain/internal/console"
	"github.com/georgealton/rain/internal/console/spinner"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// AddDefaults add standard additional flags and version information to a command
func AddDefaults(c *cobra.Command) {
	// Don't add "[flags]" to the usage line
	c.DisableFlagsInUseLine = true

	// Set version string
	c.Version = config.VERSION

	// Add the debug flag
	c.PersistentFlags().BoolVarP(&config.Debug, "debug", "", false, "Output debugging information")

	// Customise version string
	if c.Name() == "rain" {
		c.SetVersionTemplate(fmt.Sprintf("%s {{.Version}} %s/%s\n",
			config.NAME,
			runtime.GOOS,
			runtime.GOARCH,
		))
	} else {
		c.SetVersionTemplate(fmt.Sprintf("{{.Name}} (%s {{.Version}} %s/%s)\n",
			config.NAME,
			runtime.GOOS,
			runtime.GOARCH,
		))
	}
}

// Wrap creates a new command with the same functionality as src
// but with a new name and default options added for executables
// e.g. the --debug flag
// The new command is then executed
func Wrap(name string, src *cobra.Command) {
	use := strings.Split(src.Use, " ")
	use[0] = name

	// Create the new command
	out := &cobra.Command{
		Use:   strings.Join(use, " "),
		Short: src.Short,
		Long:  src.Long,
		Args:  src.Args,
		Run:   src.Run,
	}

	// Set default options
	AddDefaults(out)

	// Add the flags
	src.Flags().VisitAll(func(f *pflag.Flag) {
		out.Flags().AddFlag(f)
	})

	Execute(out)
}

func execute(cmd *cobra.Command) (code int) {
	defer func() {
		spinner.Stop()

		if r := recover(); r != nil {
			if config.Debug {
				panic(r)
			}

			fmt.Fprintln(os.Stderr, console.Red(fmt.Sprint(r)))

			code = 1
		}
	}()

	if err := cmd.Execute(); err != nil {
		code = 1
	}

	return
}

// Execute wraps a command with error trapping that deals with the debug flag
func Execute(cmd *cobra.Command) {
	os.Exit(execute(cmd))
}

// Test runs a command without calling os.Exit and instead returning the error code.
// Use this in place of Execute for functional testing
func Test(cmd *cobra.Command) int {
	return execute(cmd)
}
