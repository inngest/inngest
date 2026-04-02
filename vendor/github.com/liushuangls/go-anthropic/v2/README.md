# go-anthropic

[![Go Reference](https://pkg.go.dev/badge/github.com/liushuangls/go-anthropic/v2.svg)](https://pkg.go.dev/github.com/liushuangls/go-anthropic/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/liushuangls/go-anthropic/v2)](https://goreportcard.com/report/github.com/liushuangls/go-anthropic/v2)
[![codecov](https://codecov.io/gh/liushuangls/go-anthropic/graph/badge.svg?token=O6JSAOZORX)](https://codecov.io/gh/liushuangls/go-anthropic)
[![Sanity check](https://github.com/liushuangls/go-anthropic/actions/workflows/pr.yml/badge.svg)](https://github.com/liushuangls/go-anthropic/actions/workflows/pr.yml)

Anthropic Claude API wrapper for Go (Unofficial).

This package has support for:
- Completions
- Streaming Completions
- Messages
- Streaming Messages
- Message Batching
- Vision and PDFs
- Tool use (with [computer use](https://docs.anthropic.com/en/docs/build-with-claude/computer-use))
- Prompt Caching
- Token Counting

## Installation

```
go get github.com/liushuangls/go-anthropic/v2
```

Currently, go-anthropic requires Go version 1.21 or greater.

## Usage

### Messages example usage:

```go
package main

import (
	"errors"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
)

func main() {
	client := anthropic.NewClient("your anthropic api key")
	resp, err := client.CreateMessages(context.Background(), anthropic.MessagesRequest{
		Model: anthropic.ModelClaude3Haiku20240307,
		Messages: []anthropic.Message{
			anthropic.NewUserTextMessage("What is your name?"),
		},
		MaxTokens: 1000,
	})
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages error: %v\n", err)
		}
		return
	}
	fmt.Println(resp.Content[0].GetText())
}
```

### Messages stream example usage:

```go
package main

import (
	"errors"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
)

func main() {
	client := anthropic.NewClient("your anthropic api key")
	resp, err := client.CreateMessagesStream(context.Background(), anthropic.MessagesStreamRequest{
		MessagesRequest: anthropic.MessagesRequest{
			Model: anthropic.ModelClaude3Haiku20240307,
			Messages: []anthropic.Message{
				anthropic.NewUserTextMessage("What is your name?"),
			},
			MaxTokens: 1000,
		},
		OnContentBlockDelta: func(data anthropic.MessagesEventContentBlockDeltaData) {
			fmt.Printf("Stream Content: %s\n", data.Delta.Text)
		},
	})
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages stream error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages stream error: %v\n", err)
		}
		return
	}
	fmt.Println(resp.Content[0].GetText())
}
```

### Other examples:

<details>
<summary>Messages Vision example</summary>

```go
package main

import (
	"errors"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
)

func main() {
	client := anthropic.NewClient("your anthropic api key")

	imagePath := "xxx"
	imageMediaType := "image/jpeg"
	imageFile, err := os.Open(imagePath)
	if err != nil {
		panic(err)
	}
	imageData, err := io.ReadAll(imageFile)
	if err != nil {
		panic(err)
	}

	resp, err := client.CreateMessages(context.Background(), anthropic.MessagesRequest{
		Model: anthropic.ModelClaude3Opus20240229,
		Messages: []anthropic.Message{
			{
				Role: anthropic.RoleUser,
				Content: []anthropic.MessageContent{
					anthropic.NewImageMessageContent(
						anthropic.NewMessageContentSource(
							anthropic.MessagesContentSourceTypeBase64,
							imageMediaType,
							imageData,
						),
					),
					anthropic.NewTextMessageContent("Describe this image."),
				},
			},
		},
		MaxTokens: 1000,
	})
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages error: %v\n", err)
		}
		return
	}
	fmt.Println(*resp.Content[0].GetText())
}
```
</details>

<details>
<summary>Messages Tool use example</summary>

```go
package main

import (
	"context"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/liushuangls/go-anthropic/v2/jsonschema"
)

func main() {
	client := anthropic.NewClient(
		"your anthropic api key",
	)

	request := anthropic.MessagesRequest{
		Model: anthropic.ModelClaude3Haiku20240307,
		Messages: []anthropic.Message{
			anthropic.NewUserTextMessage("What is the weather like in San Francisco?"),
		},
		MaxTokens: 1000,
		Tools: []anthropic.ToolDefinition{
			{
				Name:        "get_weather",
				Description: "Get the current weather in a given location",
				InputSchema: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"location": {
							Type:        jsonschema.String,
							Description: "The city and state, e.g. San Francisco, CA",
						},
						"unit": {
							Type:        jsonschema.String,
							Enum:        []string{"celsius", "fahrenheit"},
							Description: "The unit of temperature, either 'celsius' or 'fahrenheit'",
						},
					},
					Required: []string{"location"},
				},
			},
		},
	}

	resp, err := client.CreateMessages(context.Background(), request)
	if err != nil {
		panic(err)
	}

	request.Messages = append(request.Messages, anthropic.Message{
		Role:    anthropic.RoleAssistant,
		Content: resp.Content,
	})

	var toolUse *anthropic.MessageContentToolUse

	for _, c := range resp.Content {
		if c.Type == anthropic.MessagesContentTypeToolUse {
			toolUse = c.MessageContentToolUse
		}
	}

	if toolUse == nil {
		panic("tool use not found")
	}

	request.Messages = append(request.Messages, anthropic.NewToolResultsMessage(toolUse.ID, "65 degrees", false))

	resp, err = client.CreateMessages(context.Background(), request)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Response: %+v\n", resp)
}
```
</details>

<details>
<summary>Prompt Caching</summary>

doc: https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching

```go
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
)

func main() {
	client := anthropic.NewClient(
		"your anthropic api key",
		anthropic.WithBetaVersion(anthropic.BetaPromptCaching20240731),
	)

	resp, err := client.CreateMessages(
		context.Background(),
		anthropic.MessagesRequest{
			Model: anthropic.ModelClaude3Haiku20240307,
			MultiSystem: []anthropic.MessageSystemPart{
				{
					Type: "text",
					Text: "You are an AI assistant tasked with analyzing literary works. Your goal is to provide insightful commentary on themes, characters, and writing style.",
				},
				{
					Type: "text",
					Text: "<the entire contents of Pride and Prejudice>",
					CacheControl: &anthropic.MessageCacheControl{
						Type: anthropic.CacheControlTypeEphemeral,
					},
				},
			},
			Messages: []anthropic.Message{
				anthropic.NewUserTextMessage("Analyze the major themes in Pride and Prejudice.")
			},
			MaxTokens: 1000,
	})
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages error: %v\n", err)
		}
		return
	}
	fmt.Printf("Usage: %+v\n", resp.Usage)
	fmt.Println(resp.Content[0].GetText())
}
```
</details>

<details>
<summary>VertexAI example</summary>
If you are using a Google Credentials file, you can use the following code to create a client:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/liushuangls/go-anthropic/v2"
	"golang.org/x/oauth2/google"
)

func main() {
	credBytes, err := os.ReadFile("<path to your credentials file>")
	if err != nil {
		fmt.Println("Error reading file")
		return
	}

	ts, err := google.JWTAccessTokenSourceWithScope(credBytes, "https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/cloud-platform.read-only")
	if err != nil {
		fmt.Println("Error creating token source")
		return
	}

	// use JWTAccessTokenSourceWithScope
	token, err := ts.Token()
	if err != nil {
		fmt.Println("Error getting token")
		return
	}

	fmt.Println(token.AccessToken)

	client := anthropic.NewClient(token.AccessToken, anthropic.WithVertexAI("<YOUR PROJECTID>", "<YOUR LOCATION>"))

	resp, err := client.CreateMessagesStream(context.Background(), anthropic.MessagesStreamRequest{
		MessagesRequest: anthropic.MessagesRequest{
			Model: anthropic.ModelClaude3Haiku20240307,
			Messages: []anthropic.Message{
				anthropic.NewUserTextMessage("What is your name?"),
			},
			MaxTokens: 1000,
		},
		OnContentBlockDelta: func(data anthropic.MessagesEventContentBlockDeltaData) {
			fmt.Printf("Stream Content: %s\n", *data.Delta.Text)
		},
	})
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages stream error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages stream error: %v\n", err)
		}
		return
	}
	fmt.Println(resp.Content[0].GetText())
}
```
</details>

<details>
<summary>Message Batching</summary>

doc: https://docs.anthropic.com/en/docs/build-with-claude/message-batches

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/liushuangls/go-anthropic/v2"
)

func main() {
	client := anthropic.NewClient(
		"your anthropic api key",
		anthropic.WithBetaVersion(anthropic.BetaMessageBatches20240924),
	)

	resp, err := client.CreateBatch(context.Background(),
		anthropic.BatchRequest{
			Requests: []anthropic.InnerRequests{
				{
					CustomId: myId,
					Params: anthropic.MessagesRequest{
						Model: anthropic.ModelClaude3Haiku20240307,
						MultiSystem: anthropic.NewMultiSystemMessages(
							"you are an assistant",
							"you are snarky",
						),
						MaxTokens: 10,
						Messages: []anthropic.Message{
							anthropic.NewUserTextMessage("What is your name?"),
							anthropic.NewAssistantTextMessage("My name is Claude."),
							anthropic.NewUserTextMessage("What is your favorite color?"),
						},
					},
				},
			},
		},
	)
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages error: %v\n", err)
		}
		return
	}
	fmt.Println(resp)


	retrieveResp, err := client.RetrieveBatch(ctx, resp.Id)
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages error: %v\n", err)
		}
		return
	}
	fmt.Println(retrieveResp)

	resultResp, err := client.RetrieveBatchResults(ctx, "batch_id_your-batch-here")
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages error: %v\n", err)
		}
		return
	}
	fmt.Println(resultResp)


	listResp, err := client.ListBatches(ctx, anthropic.ListBatchesRequest{})
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			fmt.Printf("Messages error, type: %s, message: %s", e.Type, e.Message)
		} else {
			fmt.Printf("Messages error: %v\n", err)
		}
		return
	}
	fmt.Println(listResp)


	cancelResp, err := client.CancelBatch(ctx, "batch_id_your-batch-here")
	if err != nil {
		t.Fatalf("CancelBatch error: %s", err)
	}
	fmt.Println(cancelResp)
```
</details>

<details>
<summary>Token Counting example</summary>

doc: (https://docs.anthropic.com/en/docs/build-with-claude/token-counting)[https://docs.anthropic.com/en/docs/build-with-claude/token-counting]

```go
// TODO: add example!
```

</details>

### Beta features
Anthropic provides several beta features that can be enabled using the following beta version identifiers:

Beta Version Identifier            | Code Constant                   | Description
-----------------------------------|---------------------------------|---------------------------------
`tools-2024-04-04`                 | `BetaTools20240404`             | Initial tools beta
`tools-2024-05-16`                 | `BetaTools20240516`             | Updated tools beta
`prompt-caching-2024-07-31`        | `BetaPromptCaching20240731`     | Prompt caching beta
`message-batches-2024-09-24`       | `BetaMessageBatches20240924`    | Message batching beta
`token-counting-2024-11-01`        | `BetaTokenCounting20241101`     | Token counting beta
`max-tokens-3-5-sonnet-2024-07-15` | `BetaMaxTokens35Sonnet20240715` | Max tokens beta for Sonnet model
`computer-use-2024-10-22`          | `BetaComputerUse20241022`       | Computer use beta

### Supported models
The following models are supported by go-anthropic.
These models are also available for use on Google's Vertex AI platform as well.

Model Name                     | Model String
-------------------------------|------------------------------
ModelClaude2Dot0               | "claude-2.0"
ModelClaude2Dot1               | "claude-2.1"
ModelClaude3Opus20240229       | "claude-3-opus-20240229"
ModelClaude3Sonnet20240229     | "claude-3-sonnet-20240229"
ModelClaude3Dot5Sonnet20240620 | "claude-3-5-sonnet-20240620"
ModelClaude3Dot5Sonnet20241022 | "claude-3-5-sonnet-20241022"
ModelClaude3Dot5SonnetLatest   | "claude-3-5-sonnet-latest"
ModelClaude3Haiku20240307      | "claude-3-haiku-20240307"
ModelClaude3Dot5HaikuLatest    | "claude-3-5-haiku-latest"
ModelClaude3Dot5Haiku20241022  | "claude-3-5-haiku-20241022"

### Other Enums
Two exported enums are additionally provided:
- `RoleUser` = "user": Input role type for user messages
- `RoleAssistant` = "assistant": Input role type for assistant/Claude messages

## Acknowledgments
The following project had particular influence on go-anthropic's design.

- [sashabaranov/go-openai](https://github.com/sashabaranov/go-openai)

Additionally, we thank anthropic for providing the API and documentation.

## License
go-anthropic is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.
