# OpenAI

OpenAI client wrapper for AI capabilities in your Go applications. Built on [openai-go/v3](https://github.com/openai/openai-go).

## Installation

```bash
go get github.com/dotcommander/gokart
```

## Quick Start

```go
import "github.com/dotcommander/gokart"

// Create client with default environment variable
client := gokart.NewOpenAIClient()

// Create client with explicit API key
client := gokart.NewOpenAIClientWithKey("sk-...")

// Basic completion
ctx := context.Background()
completion, _ := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("Hello, GoKart!"),
    }),
    Model: openai.F(openai.ChatModelGPT4o),
})
```

---

## Configuration

### Environment Variable

The default client reads from the `OPENAI_API_KEY` environment variable:

```bash
export OPENAI_API_KEY="sk-..."
```

### API Key Configuration

#### Using Environment Variable (Recommended)

```go
client := gokart.NewOpenAIClient()
```

#### Using Explicit API Key

```go
client := gokart.NewOpenAIClientWithKey("sk-...")
```

---

## Basic Usage

### Simple Chat Completion

```go
ctx := context.Background()

completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("What is GoKart?"),
    }),
    Model: openai.F(openai.ChatModelGPT4o),
})

if err != nil {
    log.Fatal(err)
}

fmt.Println(completion.Choices[0].Message.Content)
```

### System Messages

```go
completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
        openai.SystemMessage("You are a helpful assistant specializing in Go programming."),
        openai.UserMessage("How do I read a file in Go?"),
    }),
    Model: openai.F(openai.ChatModelGPT4o),
})
```

### Conversation History

```go
messages := []openai.ChatCompletionMessageParamUnion{
    openai.UserMessage("What's the capital of France?"),
    openai.AssistantMessage("The capital of France is Paris."),
    openai.UserMessage("And what's its population?"),
}

completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Messages: openai.F(messages),
    Model:    openai.F(openai.ChatModelGPT4o),
})
```

---

## Streaming Responses

```go
stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
    Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("Tell me a short story"),
    }),
    Model: openai.F(openai.ChatModelGPT4o),
})

for stream.Next() {
    chunk := stream.Current()
    fmt.Print(chunk.Choices[0].Delta.Content)
}

if err := stream.Err(); err != nil {
    log.Fatal(err)
}
```

---

## Models

### Available Models

```go
// GPT-4o (recommended for most use cases)
Model: openai.F(openai.ChatModelGPT4o)

// GPT-4o-mini (faster, lower cost)
Model: openai.F(openai.ChatModelGPT4oMini)

// GPT-4 Turbo
Model: openai.F(openai.ChatModelGPT4Turbo)

// GPT-3.5 Turbo (legacy)
Model: openai.F(openai.ChatModelGPT35Turbo)
```

### Model Selection Guide

| Model | Use Case | Cost |
|-------|----------|------|
| `GPT4o` | General purpose, complex reasoning | Higher |
| `GPT4oMini` | Simple tasks, high volume | Lower |
| `GPT4Turbo` | Legacy applications | Medium |
| `GPT35Turbo` | Cost-sensitive legacy apps | Lowest |

---

## Reference

### Functions

| Function | Description |
|----------|-------------|
| `NewOpenAIClient` | Creates client using `OPENAI_API_KEY` env var |
| `NewOpenAIClientWithKey` | Creates client with explicit API key |

### Function Signatures

```go
// Creates OpenAI client with environment variable
func NewOpenAIClient(opts ...option.RequestOption) openai.Client

// Creates OpenAI client with explicit API key
func NewOpenAIClientWithKey(apiKey string) openai.Client
```

### See Also

- [OpenAI Go SDK documentation](https://github.com/openai/openai-go)
- [OpenAI API reference](https://platform.openai.com/docs/api-reference)
- [HTTP client](/api/gokart#http-client) - Retryable HTTP client
- [Response helpers](/components/response) - JSON response helpers
