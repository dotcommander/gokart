// Example: OpenAI API integration with GoKart.
//
// This example demonstrates:
//   - Creating an OpenAI client (uses OPENAI_API_KEY env var)
//   - Chat completions with messages
//   - Streaming responses
//   - Using different models
//
// Prerequisites:
//   - Set OPENAI_API_KEY environment variable:
//     export OPENAI_API_KEY="sk-..."
//
// Run with: go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dotcommander/gokart"
	"github.com/openai/openai-go/v3"
)

func main() {
	ctx := context.Background()

	// Check for API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	// Example 1: Create client using environment variable (recommended)
	// The SDK automatically reads OPENAI_API_KEY from the environment
	client := gokart.NewOpenAIClient()

	// Example 2: Simple chat completion
	log.Println("Sending chat completion request...")

	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("What is the capital of France? Answer in one sentence."),
		},
		Model: openai.ChatModelGPT4oMini,
	})
	if err != nil {
		log.Fatalf("Chat completion failed: %v", err)
	}

	if len(chatCompletion.Choices) > 0 {
		fmt.Printf("Response: %s\n", chatCompletion.Choices[0].Message.Content)
	}

	// Example 3: Multi-turn conversation
	log.Println("\nMulti-turn conversation:")

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful assistant that gives concise answers."),
		openai.UserMessage("What's 2 + 2?"),
	}

	response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    openai.ChatModelGPT4oMini,
	})
	if err != nil {
		log.Printf("Conversation failed: %v", err)
	} else {
		assistantReply := response.Choices[0].Message.Content
		fmt.Printf("Assistant: %s\n", assistantReply)

		// Continue the conversation
		messages = append(messages,
			openai.AssistantMessage(assistantReply),
			openai.UserMessage("Now multiply that by 10"),
		)

		response, err = client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4oMini,
		})
		if err != nil {
			log.Printf("Follow-up failed: %v", err)
		} else {
			fmt.Printf("Assistant: %s\n", response.Choices[0].Message.Content)
		}
	}

	// Example 4: Streaming response
	log.Println("\nStreaming response:")

	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Count from 1 to 5, with a brief pause between each number."),
		},
		Model: openai.ChatModelGPT4oMini,
	})

	fmt.Print("Streaming: ")
	for stream.Next() {
		event := stream.Current()
		if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
			fmt.Print(event.Choices[0].Delta.Content)
		}
	}
	fmt.Println()

	if err := stream.Err(); err != nil {
		log.Printf("Stream error: %v", err)
	}

	// Example 5: Using different models
	log.Println("\nUsing GPT-4o:")

	gpt4Response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Explain quantum entanglement in one sentence."),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		log.Printf("GPT-4o request failed: %v", err)
	} else {
		fmt.Printf("GPT-4o: %s\n", gpt4Response.Choices[0].Message.Content)
	}

	// Example 6: With explicit API key
	// Useful when managing multiple API keys or testing
	// keyClient := gokart.NewOpenAIClientWithKey("sk-...")
	// keyClient.Chat.Completions.New(ctx, params)

	log.Println("\nDone!")
}

// Example: Temperature and other parameters
//
// Control response randomness and length:
//
//	response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
//	    Messages: messages,
//	    Model:    openai.ChatModelGPT4oMini,
//	    Temperature: openai.Float(0.7),  // 0.0 = deterministic, 2.0 = very random
//	    MaxTokens:   openai.Int(100),    // Limit response length
//	    TopP:        openai.Float(0.9),  // Nucleus sampling
//	})

// Example: JSON mode
//
// Force structured JSON output:
//
//	response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
//	    Messages: []openai.ChatCompletionMessageParamUnion{
//	        openai.SystemMessage("You respond in JSON format only."),
//	        openai.UserMessage("List 3 programming languages with their year of creation."),
//	    },
//	    Model:          openai.ChatModelGPT4oMini,
//	    ResponseFormat: openai.ChatCompletionNewParamsResponseFormatJSONObject,
//	})

// Example: Function calling
//
// Let the model call your functions:
//
//	tools := []openai.ChatCompletionToolParam{
//	    {
//	        Type: openai.ChatCompletionToolTypeFunction,
//	        Function: openai.FunctionDefinitionParam{
//	            Name:        "get_weather",
//	            Description: openai.String("Get current weather for a location"),
//	            Parameters: openai.FunctionParameters{
//	                "type": "object",
//	                "properties": map[string]any{
//	                    "location": map[string]string{
//	                        "type":        "string",
//	                        "description": "City name",
//	                    },
//	                },
//	                "required": []string{"location"},
//	            },
//	        },
//	    },
//	}
//
//	response, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
//	    Messages: messages,
//	    Model:    openai.ChatModelGPT4oMini,
//	    Tools:    tools,
//	})
