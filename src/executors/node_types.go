package executors

// Node type constants - centralized source of truth for all node types
const (
	// Trigger node types
	NodeTypeStart = "start"

	// Processing node types
	NodeTypeConditional     = "conditional"
	NodeTypeTransform       = "transform"
	NodeTypeDelay           = "delay"
	NodeTypeEmail           = "email"
	NodeTypeHTTP            = "http"
	NodeTypeSlack           = "slack"
	NodeTypeLoop            = "loop"
	NodeTypeLoopAccumulator = "loop-accumulator"
	NodeTypeJSON            = "json"
	NodeTypeJSONArray       = "json-array"
	NodeTypeJSONToCSV       = "json_to_csv"

	// End node types
	NodeTypeEnd = "end"
)

// Node type categories
var (
	// TriggerNodeTypes are node types that start workflow execution
	TriggerNodeTypes = []string{NodeTypeStart}

	// EndNodeTypes are node types that mark workflow completion
	EndNodeTypes = []string{NodeTypeEnd}

	// AsyncNodeTypes are node types that require the outbox pattern
	AsyncNodeTypes = []string{NodeTypeEmail, NodeTypeSlack}

	// AllValidNodeTypes contains all supported node types for validation
	AllValidNodeTypes = []string{
		NodeTypeStart,
		NodeTypeConditional,
		NodeTypeTransform,
		NodeTypeDelay,
		NodeTypeEmail,
		NodeTypeHTTP,
		NodeTypeSlack,
		NodeTypeLoop,
		NodeTypeLoopAccumulator,
		NodeTypeJSON,
		NodeTypeJSONArray,
		NodeTypeJSONToCSV,
		NodeTypeEnd,
	}
)

// IsSkippableNode returns true if node should not be executed
// (start and end nodes are traversed but not executed)
func IsSkippableNode(nodeType string) bool {
	return nodeType == NodeTypeStart || nodeType == NodeTypeEnd
}

// IsAsyncNode returns true if node requires outbox pattern
func IsAsyncNode(nodeType string) bool {
	for _, t := range AsyncNodeTypes {
		if t == nodeType {
			return true
		}
	}
	return false
}

// IsValidNodeType returns true if the node type is supported
func IsValidNodeType(nodeType string) bool {
	for _, t := range AllValidNodeTypes {
		if t == nodeType {
			return true
		}
	}
	return false
}

// IsTriggerNode returns true if the node type is a trigger node
func IsTriggerNode(nodeType string) bool {
	for _, t := range TriggerNodeTypes {
		if t == nodeType {
			return true
		}
	}
	return false
}

// IsEndNode returns true if the node type is an end node
func IsEndNode(nodeType string) bool {
	for _, t := range EndNodeTypes {
		if t == nodeType {
			return true
		}
	}
	return false
}
