package anthropic

// VertexAISupport is an interface that is used to configure Vertex AI requests.
// The changes are defined here: https://docs.anthropic.com/en/api/claude-on-vertex-ai
// Model needs to be in the calling URL
// The version of the API is defined in the request body
// This interface allows the vertex ai changes to be contained in the client code, and not leak to each indivdual request definition.
type VertexAISupport interface {
	GetModel() Model
	SetAnthropicVersion(APIVersion)
	IsStreaming() bool
}
