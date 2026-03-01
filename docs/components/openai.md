# OpenAI

OpenAI client wrapper for AI capabilities in your Go applications. Built on [openai-go/v3](https://github.com/openai/openai-go).

## Installation

```bash
go get github.com/dotcommander/gokart/ai
```

## Quick Start

```go
import "github.com/dotcommander/gokart/ai"

// Create client with default environment variable
client := ai.NewOpenAIClient()

// Create client with explicit API key
client := ai.NewOpenAIClientWithKey("sk-...")

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
client := ai.NewOpenAIClient()
```

#### Using Explicit API Key

```go
client := ai.NewOpenAIClientWithKey("sk-...")
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

## Best Practices

### Structured Output

Use `ResponseFormatJSONObject` to request machine-readable JSON responses. The model is instructed to always return valid JSON, which you can then unmarshal directly:

```go
completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
        openai.SystemMessage("Extract the fields as JSON. Return {\"name\": string, \"age\": number}."),
        openai.UserMessage("Name: Alice, Age: 30"),
    }),
    Model: openai.F(openai.ChatModelGPT4o),
    ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
        openai.ResponseFormatJSONObjectParam{
            Type: openai.F(openai.ResponseFormatJSONObjectTypeJSONObject),
        },
    ),
})
if err != nil {
    return err
}

var result struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}
if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &result); err != nil {
    return fmt.Errorf("unmarshal response: %w", err)
}
```

### Streaming Long Responses

For user-facing output, prefer streaming over blocking. It writes tokens as they arrive, which reduces perceived latency significantly for long responses:

```go
stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
    Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("Summarize this document: " + doc),
    }),
    Model: openai.F(openai.ChatModelGPT4o),
})

for stream.Next() {
    chunk := stream.Current()
    if len(chunk.Choices) > 0 {
        fmt.Print(chunk.Choices[0].Delta.Content)
    }
}

if err := stream.Err(); err != nil {
    return fmt.Errorf("stream: %w", err)
}
```

### Error Handling

The SDK returns typed errors. Distinguish API-level errors (invalid request, quota exceeded, rate limit) from transport errors (network timeout, context cancellation) so you can surface useful messages:

```go
completion, err := client.Chat.Completions.New(ctx, params)
if err != nil {
    var apiErr *openai.Error
    if errors.As(err, &apiErr) {
        // HTTP-level error from the API: check StatusCode and Message
        return fmt.Errorf("openai error %d: %s", apiErr.StatusCode, apiErr.Message)
    }
    // Network or context error (e.g., context.DeadlineExceeded)
    return fmt.Errorf("openai request: %w", err)
}

if len(completion.Choices) == 0 {
    return errors.New("no completion choices returned")
}
```

### Reuse the Client

`ai.NewOpenAIClient()` returns an `openai.Client` that is safe to share across requests and goroutines. Create it once — in your app context or command setup — and inject it wherever it is needed. Calling the constructor per request adds unnecessary overhead and bypasses the connection pool built into the underlying HTTP client.

### Model Constants

Prefer the SDK-provided model constants over raw string literals. The constants are defined in the `openai` package alongside the rest of the API surface. See the [Models](#models) section above for the available constants and a guide on when to use each.

If you need a model that does not yet have a constant (e.g., a preview release), you can cast a string directly:

```go
Model: openai.F(openai.ChatModel("the-model-id"))
```

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

**Upstream SDK:**
- [openai-go on pkg.go.dev](https://pkg.go.dev/github.com/openai/openai-go/v3) — full API reference for the underlying SDK
- [openai-go on GitHub](https://github.com/openai/openai-go) — source, changelog, and examples
- [OpenAI API reference](https://platform.openai.com/docs/api-reference) — REST API documentation and model capabilities

**GoKart components:**
- [CLI](/api/cli) — build AI-powered CLI tools using the same client alongside commands, tables, and spinners
- [Config](/api/gokart) — load `OPENAI_API_KEY` and model configuration from config files or environment
- [Web](/components/web) — build AI-powered HTTP APIs with the [HTTP client](/components/web#http-client) and [Response helpers](/components/response) to return completions as JSON
- [Cache](/components/cache) — cache expensive completion results with the Remember pattern to reduce API calls
