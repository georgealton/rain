package watch

import (
	"errors"
	"fmt"
	"time"

	"github.com/georgealton/rain/internal/ui"

	"github.com/georgealton/rain/internal/aws/cfn"
	"github.com/georgealton/rain/internal/console"
	"github.com/georgealton/rain/internal/console/spinner"
	"github.com/spf13/cobra"
)

var waitThenWatch = false

// Cmd is the watch command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "watch <stack>",
	Short:                 "Display an updating view of a CloudFormation stack",
	Long:                  "Repeatedly displays the status of a CloudFormation stack. Useful for watching the progress of a deployment started from outside of Rain.",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		first := true
		for {
			if first {
				spinner.Push("Fetching stack status")
			}

			stack, err := cfn.GetStack(stackName)
			if err != nil {
				panic(ui.Errorf(err, "error watching stack '%s'", stackName))
			}

			if !cfn.StackHasSettled(stack) {
				// Stack is changing
				break
			}

			if !waitThenWatch {
				// Not changing, not waiting for it
				status, _ := cfn.GetStackOutput(stack)
				fmt.Println(status)
				panic(errors.New("not watching unchanging stack"))
			}

			if first {
				spinner.Pop()
				spinner.Push("Waiting for stack to begin changing")
				first = false
			}

			time.Sleep(time.Second * cfn.WaitPeriodInSeconds)
		}

		spinner.Pop()

		status, messages := cfn.WaitForStackToSettle(stackName)

		fmt.Println("Final stack status:", ui.ColouriseStatus(status))

		if len(messages) > 0 {
			fmt.Println(console.Yellow("Messages:"))
			for _, message := range messages {
				fmt.Printf("  - %s\n", message)
			}
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&waitThenWatch, "wait", "w", false, "wait for changes to begin rather than refusing to watch an unchanging stack")
}
