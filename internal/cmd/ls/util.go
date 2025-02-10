package ls

import (
	"fmt"
	"strings"

	"github.com/georgealton/rain/internal/ui"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func formatStack(stack types.StackSummary, stackMap map[string]types.StackSummary) string {
	out := strings.Builder{}

	out.WriteString(fmt.Sprintf("%s: %s\n",
		*stack.StackName,
		ui.ColouriseStatus(string(stack.StackStatus)),
	))

	for _, otherStack := range stackMap {
		if otherStack.ParentId != nil && *otherStack.ParentId == *stack.StackId {
			out.WriteString(ui.Indent("  - ", formatStack(otherStack, stackMap)))
			out.WriteString("\n")
		}
	}

	return out.String()
}
