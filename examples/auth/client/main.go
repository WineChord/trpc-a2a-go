// Tencent is pleased to support the open source community by making trpc-a2a-go available.
//
// Copyright (C) 2025 THL A29 Limited, a Tencent company.  All rights reserved.
//
// trpc-a2a-go is licensed under the Apache License Version 2.0.

// Package main provides example code for using different authentication methods with the A2A client.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"trpc.group/trpc-go/trpc-a2a-go/auth"
	"trpc.group/trpc-go/trpc-a2a-go/client"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: auth_client <auth_method> [options]")
		fmt.Println("Auth methods: jwt, apikey, oauth2")
		return
	}

	authMethod := os.Args[1]
	agentURL := "http://localhost:8080/"

	var a2aClient *client.A2AClient
	var err error

	// Create client with the specified authentication method
	switch authMethod {
	case "jwt":
		a2aClient, err = createJWTClient(agentURL)
	case "apikey":
		a2aClient, err = createAPIKeyClient(agentURL)
	case "oauth2":
		a2aClient, err = createOAuth2Client(agentURL)
	default:
		fmt.Printf("Unknown authentication method: %s\n", authMethod)
		return
	}

	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create a simple task to test authentication
	textPart := protocol.NewTextPart("Hello, this is an authenticated request")
	message := protocol.NewMessage(protocol.MessageRoleUser, []protocol.Part{textPart})

	// Send the task
	task, err := a2aClient.SendTasks(context.Background(), protocol.SendTaskParams{
		ID:      "auth-test-task",
		Message: message,
	})

	if err != nil {
		log.Fatalf("Failed to send task: %v", err)
	}

	fmt.Printf("Task ID: %s, Status: %s\n", task.ID, task.Status.State)

	// For demonstration purposes, get the task status
	taskQuery := protocol.TaskQueryParams{
		ID: task.ID,
	}

	updatedTask, err := a2aClient.GetTasks(context.Background(), taskQuery)
	if err != nil {
		log.Fatalf("Failed to get task: %v", err)
	}

	fmt.Printf("Updated task status: %s\n", updatedTask.Status.State)
}

// createJWTClient creates an A2A client with JWT authentication.
func createJWTClient(agentURL string) (*client.A2AClient, error) {
	// In a real application, you would get these securely from environment variables or a key management system
	secret := []byte("my-secret-key")
	audience := "a2a-client-example"
	issuer := "example-issuer"

	return client.NewA2AClient(
		agentURL,
		client.WithJWTAuth(secret, audience, issuer, 1*time.Hour),
	)
}

// createAPIKeyClient creates an A2A client with API key authentication.
func createAPIKeyClient(agentURL string) (*client.A2AClient, error) {
	// In a real application, you would get this securely from environment variables
	apiKey := "my-api-key"
	headerName := "X-API-Key" // This is the default, but can be customized

	return client.NewA2AClient(
		agentURL,
		client.WithAPIKeyAuth(apiKey, headerName),
	)
}

// createOAuth2Client creates an A2A client with OAuth2 authentication.
func createOAuth2Client(agentURL string) (*client.A2AClient, error) {
	// Method 1: Using client credentials flow
	return createOAuth2ClientCredentialsClient(agentURL)

	// Alternative methods:
	// return createOAuth2TokenSourceClient(agentURL)
	// return createCustomOAuth2Client(agentURL)
}

// createOAuth2ClientCredentialsClient creates a client using OAuth2 client credentials flow.
func createOAuth2ClientCredentialsClient(agentURL string) (*client.A2AClient, error) {
	// In a real application, you would get these securely from environment variables
	clientID := "my-client-id"
	clientSecret := "my-client-secret"

	// Use the local OAuth2 server endpoint
	tokenURL := getOAuthTokenURL(agentURL)
	scopes := []string{"a2a.read", "a2a.write"}

	return client.NewA2AClient(
		agentURL,
		client.WithOAuth2ClientCredentials(clientID, clientSecret, tokenURL, scopes),
	)
}

// createOAuth2TokenSourceClient creates a client using a custom OAuth2 token source.
func createOAuth2TokenSourceClient(agentURL string) (*client.A2AClient, error) {
	// Extract the OAuth token URL from agentURL
	tokenURL := getOAuthTokenURL(agentURL)

	// Example with password credentials grant
	config := &oauth2.Config{
		ClientID:     "my-client-id",
		ClientSecret: "my-client-secret",
		Endpoint: oauth2.Endpoint{
			TokenURL: tokenURL,
		},
		Scopes: []string{"a2a.read", "a2a.write"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// For our mock server, we don't actually need username/password
	// as it only implements client_credentials grant
	// In a real app, these would come from configuration
	token, err := config.PasswordCredentialsToken(
		ctx,
		"username",
		"password",
	)
	if err != nil {
		return nil, fmt.Errorf("OAuth2 token acquisition failed: %w", err)
	}

	tokenSource := config.TokenSource(context.Background(), token)
	return client.NewA2AClient(
		agentURL,
		client.WithOAuth2TokenSource(config, tokenSource),
	)
}

// createCustomOAuth2Client creates a client with a completely custom OAuth2 provider.
func createCustomOAuth2Client(agentURL string) (*client.A2AClient, error) {
	// Extract the OAuth token URL from agentURL
	tokenURL := getOAuthTokenURL(agentURL)

	// Create a client credentials config
	ccConfig := &clientcredentials.Config{
		ClientID:     "my-client-id",
		ClientSecret: "my-client-secret",
		TokenURL:     tokenURL,
		Scopes:       []string{"a2a.read", "a2a.write"},
	}

	// Create a custom OAuth2 provider
	provider := auth.NewOAuth2ClientCredentialsProvider(
		ccConfig.ClientID,
		ccConfig.ClientSecret,
		ccConfig.TokenURL,
		ccConfig.Scopes,
	)

	// Use the custom provider
	return client.NewA2AClient(
		agentURL,
		client.WithAuthProvider(provider),
	)
}

// Helper function to get the OAuth token URL based on agent URL
func getOAuthTokenURL(agentURL string) string {
	tokenURL := ""
	if agentURL == "http://localhost:8080/" {
		tokenURL = "http://localhost:8080/oauth2/token"
	} else {
		// Try to adapt to a different port
		// This is a simple adaptation, not fully robust
		tokenURL = agentURL + "oauth2/token"
		if tokenURL[len(tokenURL)-1] == '/' {
			tokenURL = tokenURL[:len(tokenURL)-1]
		}
	}
	fmt.Printf("Using OAuth2 token URL: %s\n", tokenURL)
	return tokenURL
}
