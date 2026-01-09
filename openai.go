// Package gokart provides thin wrappers around battle-tested packages.
// This file provides OpenAI client factory functions.
package gokart

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// NewOpenAIClient creates an OpenAI client with the given options.
// By default, the SDK reads from the OPENAI_API_KEY environment variable.
func NewOpenAIClient(opts ...option.RequestOption) openai.Client {
	return openai.NewClient(opts...)
}

// NewOpenAIClientWithKey creates an OpenAI client with the specified API key.
func NewOpenAIClientWithKey(apiKey string) openai.Client {
	return openai.NewClient(option.WithAPIKey(apiKey))
}
