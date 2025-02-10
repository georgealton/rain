package cfn

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/georgealton/rain/internal/console"
	"github.com/georgealton/rain/internal/console/spinner"
	"github.com/georgealton/rain/internal/ui"
)

func StatusIsSettled(status string) bool {
	if strings.HasSuffix(status, "_COMPLETE") || strings.HasSuffix(status, "_FAILED") {
		return true
	}
	return false
}

// StackHasSettled returns whether a given status represents
// a stack that has settled, i.e. is not updating
func StackHasSettled(stack types.Stack) bool {
	return StatusIsSettled(string(stack.StackStatus))
}

func stackResourceStatuses(stack types.Stack) (string, []string) {
	stackName := ptr.ToString(stack.StackName)

	statuses := make(map[string]string)
	messages := make([]string, 0)
	nested := make(map[string]string)

	// Get changeset details if possible
	changeset, err := GetChangeSet(stackName, ptr.ToString(stack.ChangeSetId))
	if err == nil {
		for _, change := range changeset.Changes {
			resourceID := ptr.ToString(change.ResourceChange.LogicalResourceId)
			statuses[resourceID] = "REVIEW_IN_PROGRESS"

			// Store nested stacks
			if ptr.ToString(change.ResourceChange.ResourceType) == "AWS::CloudFormation::Stack" {
				nested[resourceID] = fmt.Sprintf("%s: %s", console.Yellow(fmt.Sprintf("Stack %s", resourceID)), console.Grey("PENDING"))
			}
		}
	}

	// We ignore errors because it just means we'll list no resources
	resources, _ := GetStackResources(stackName)
	for _, resource := range resources {
		resourceID := ptr.ToString(resource.LogicalResourceId)

		status := string(resource.ResourceStatus)
		rep := ui.MapStatus(status)

		statuses[resourceID] = status

		// Store messages
		if resource.ResourceStatusReason != nil && rep.Category == ui.Failed {
			msg := ptr.ToString(resource.ResourceStatusReason)
			colour := ui.StatusColour[rep.Category]

			if msg != "Resource creation cancelled" {
				id := resourceID
				if resource.PhysicalResourceId != nil {
					id += " - " + ptr.ToString(resource.PhysicalResourceId)
				}

				messages = append(messages, fmt.Sprintf("%s %s", console.Yellow(fmt.Sprintf("%s:", id)), colour(msg)))
			}
		}

		// Store nested stacks
		if ptr.ToString(resource.ResourceType) == "AWS::CloudFormation::Stack" {
			stack, err := GetStack(ptr.ToString(resource.PhysicalResourceId))
			if err == nil {
				rs, rMessages := GetStackOutput(stack)
				nested[resourceID] = rs
				for _, rMessage := range rMessages {
					messages = append(messages, fmt.Sprintf("%s%s", console.Yellow(fmt.Sprintf("%s/", resourceID)), rMessage))
				}
			}
		}
	}

	// Build the output
	out := strings.Builder{}
	stackStatus := string(stack.StackStatus)
	if strings.HasSuffix(stackStatus, "_IN_PROGRESS") {
		total := len(statuses)
		complete := 0
		inProgress := 0

		for _, status := range statuses {

			switch stackStatus {
			case
				"CREATE_IN_PROGRESS",
				"UPDATE_IN_PROGRESS",
				"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
				"REVIEW_IN_PROGRESS",
				"IMPORT_IN_PROGRESS":
				switch status {
				case "CREATE_COMPLETE", "CREATE_FAILED", "UPDATE_COMPLETE", "UPDATE_FAILED", "IMPORT_COMPLETE", "IMPORT_FAILED":
					complete++
				case "CREATE_IN_PROGRESS", "UPDATE_IN_PROGRESS", "IMPORT_IN_PROGRESS":
					inProgress++
				}

			case
				"DELETE_IN_PROGRESS":
				switch status {
				case "DELETE_COMPLETE", "DELETE_FAILED", "DELETE_SKIPPED":
					complete++
				case "DELETE_IN_PROGRESS":
					inProgress++
				}

			case
				"ROLLBACK_IN_PROGRESS",
				"UPDATE_ROLLBACK_IN_PROGRESS",
				"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS",
				"IMPORT_ROLLBACK_IN_PROGRESS":
				switch status {
				case "DELETE_COMPLETE", "DELETE_FAILED", "DELETE_SKIPPED", "IMPORT_ROLLBACK_COMPLETE", "IMPORT_ROLLBACK_FAILED":
					complete++
				case "DELETE_IN_PROGRESS", "IMPORT_ROLLBACK_IN_PROGRESS":
					inProgress++
				}
			}
		}

		pending := total - complete - inProgress

		parts := make([]string, 0)

		if pending > 0 {
			word := "resources"
			if pending == 1 {
				word = "resource"
			}
			parts = append(parts, console.Grey(fmt.Sprintf("%d %s pending", pending, word)))
		}

		if inProgress > 0 {
			word := "resources"
			if inProgress == 1 {
				word = "resource"
			}
			parts = append(parts, console.Blue(fmt.Sprintf("%d %s in progress", inProgress, word)))
		}

		if len(parts) > 0 {
			out.WriteString("- ")
			out.WriteString(strings.Join(parts, ", "))
		}
	}

	out.WriteString("\n")

	// Append nested stacks to the output
	names := make([]string, 0)
	for name := range nested {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		other := nested[name]
		parts := strings.Split(strings.TrimSpace(other), "\n")
		for _, part := range parts {
			out.WriteString(fmt.Sprintf("  - %s\n", part))
		}
	}

	return out.String(), messages
}

// GetStackOutput returns a pretty representation of a CloudFormation stack's status
func GetStackOutput(stack types.Stack) (string, []string) {
	out := strings.Builder{}

	stackStatus := string(stack.StackStatus)
	stackName := ptr.ToString(stack.StackName)

	rs, messages := stackResourceStatuses(stack)

	out.WriteString(fmt.Sprintf("%s: %s %s", console.Yellow(fmt.Sprintf("Stack %s", stackName)), ui.ColouriseStatus(stackStatus), rs))

	return strings.TrimSpace(out.String()), messages
}

// WaitForStackToSettle blocks excute until a stack has finished updating
// and then returns its status
func WaitForStackToSettle(stackName string) (string, []string) {
	// Start the timer
	spinner.StartTimer("")

	stackID := stackName

	collectedMessages := make(map[string]bool)

	out := strings.Builder{}
	outStr := ""
	lastOutput := ""

	for {
		out.Reset()

		stack, err := GetStack(stackID)
		if err != nil {
			panic(ui.Errorf(err, "operation failed"))
		}

		// Refresh the stack ID so we can deal with deleted stacks ok
		stackID = ptr.ToString(stack.StackId)

		output, messages := GetStackOutput(stack)

		// Send the output first
		out.WriteString(output)
		out.WriteString("\n")

		if len(messages) > 0 {
			out.WriteString(console.Yellow("Messages:\n"))
			for _, message := range messages {
				collectedMessages[message] = true
				out.WriteString(fmt.Sprintf("  - %s\n", message))
			}
		}

		outStr = out.String()

		spinner.Pause()
		console.ClearLines(console.CountLines(lastOutput))
		if console.IsTTY {
			fmt.Print(outStr)
		}
		lastOutput = outStr
		spinner.Resume()

		// Check to see if we've finished
		if StackHasSettled(stack) {
			spinner.StopTimer()

			console.ClearLines(console.CountLines(lastOutput))

			messages := make([]string, 0)
			for message := range collectedMessages {
				messages = append(messages, message)
			}

			return string(stack.StackStatus), messages
		}

		time.Sleep(time.Second * WaitPeriodInSeconds)
	}
}

// GetStackSummary returns a string representation of an existing stack.
// If long is false, only the stack status and stack outputs will be included.
// If long is true, resources and parameters will be also included in the output.
func GetStackSummary(stack types.Stack, long bool) string {
	out := strings.Builder{}

	stackStatus := string(stack.StackStatus)
	stackName := ptr.ToString(stack.StackName)

	// Stack status
	out.WriteString(fmt.Sprintf("%s: %s\n", console.Yellow(fmt.Sprintf("Stack %s", stackName)), ui.ColouriseStatus(stackStatus)))

	if long {
		// Params
		if len(stack.Parameters) > 0 {
			out.WriteString(fmt.Sprintf("  %s:\n", console.Yellow("Parameters")))
			for _, param := range stack.Parameters {
				out.WriteString(fmt.Sprintf("    %s: ", console.Yellow(ptr.ToString(param.ParameterKey))))

				if param.ResolvedValue != nil {
					out.WriteString(ptr.ToString(param.ResolvedValue))
				} else {
					out.WriteString(ptr.ToString(param.ParameterValue))
				}

				out.WriteString("\n")
			}
		}

		// Resources
		out.WriteString(fmt.Sprintf("  %s:\n", console.Yellow("Resources")))
		resources, _ := GetStackResources(stackName) // Ignore errors - it just means we'll get no resources
		for _, resource := range resources {
			out.WriteString(fmt.Sprintf("    %s: %s\n",
				console.Yellow(ptr.ToString(resource.LogicalResourceId)),
				ui.ColouriseStatus(string(resource.ResourceStatus)),
			))

			if ptr.ToString(resource.ResourceType) == "AWS::CloudFormation::Stack" {
				nestedStack, err := GetStack(ptr.ToString(resource.PhysicalResourceId))
				if err == nil {
					nestedSummary := GetStackSummary(nestedStack, long)

					for _, line := range strings.Split(nestedSummary, "\n") {
						out.WriteString(fmt.Sprintf("      %s\n", line))
					}
				}
			} else {
				out.WriteString(fmt.Sprintf("      %s\n", ptr.ToString(resource.PhysicalResourceId)))
			}
		}
	}

	// Outputs
	if len(stack.Outputs) > 0 {
		out.WriteString(fmt.Sprintf("%s:\n", console.Yellow("  Outputs")))
		for _, output := range stack.Outputs {
			out.WriteString(fmt.Sprintf("    %s: %s", console.Yellow(ptr.ToString(output.OutputKey)), ptr.ToString(output.OutputValue)))

			if output.Description != nil || output.ExportName != nil {
				out.WriteString(console.Grey(" # "))

				if output.Description != nil {
					out.WriteString(console.Grey(ptr.ToString(output.Description)))
				}

				if output.ExportName != nil {
					msg := fmt.Sprintf("exported as %s", ptr.ToString(output.ExportName))

					if output.Description != nil {
						msg = " (" + msg + ")"
					}

					out.WriteString(console.Grey(msg))
				}
			}

			out.WriteString("\n")
		}
	}

	return strings.TrimSpace(out.String())
}

func GetStackSetSummary(stackSet *types.StackSet, long bool) string {
	out := strings.Builder{}

	stackSetStatus := string(stackSet.Status)
	stackSetName := ptr.ToString(stackSet.StackSetName)

	// Stack status
	out.WriteString(fmt.Sprintf("%s: %s %s\n", console.Yellow("StackSet"), stackSetName, ui.ColouriseStatus(stackSetStatus)))

	if long {
		// Params
		if len(stackSet.Parameters) > 0 {
			out.WriteString(fmt.Sprintf("  %s:\n", console.Yellow("Parameters")))
			for _, param := range stackSet.Parameters {
				out.WriteString(fmt.Sprintf("    %s: ", console.Yellow(ptr.ToString(param.ParameterKey))))

				if param.ResolvedValue != nil {
					out.WriteString(ptr.ToString(param.ResolvedValue))
				} else {
					out.WriteString(ptr.ToString(param.ParameterValue))
				}

				out.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(out.String())
}
