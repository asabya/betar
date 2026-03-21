package sdk_test

import (
	"context"
	"fmt"

	"github.com/asabya/betar/pkg/sdk"
	"github.com/asabya/betar/pkg/types"
)

func Example() {
	ctx := context.Background()

	// Create a client — connects to the P2P network automatically.
	client, err := sdk.NewClient(sdk.Config{
		DataDir:      "/tmp/betar-example",
		GoogleAPIKey: "your-api-key",
	})
	if err != nil {
		fmt.Println("failed to create client:", err)
		return
	}
	defer client.Close()

	// Register an agent on the marketplace.
	_, err = client.Register(ctx, sdk.AgentSpec{
		Name:        "summarizer",
		Description: "Summarizes text documents",
		Price:       0.01, // 0.01 USDC per task
		Services:    []types.Service{{Name: "summarize", Version: "1.0"}},
		X402Support: true,
	})
	if err != nil {
		fmt.Println("failed to register:", err)
		return
	}

	// Discover agents on the network.
	listings, err := client.Discover(ctx, "summarizer")
	if err != nil {
		fmt.Println("failed to discover:", err)
		return
	}

	// Execute a discovered agent — x402 payment handled automatically.
	if len(listings) > 0 {
		output, err := client.Execute(ctx, listings[0].ID, "Summarize this document...")
		if err != nil {
			fmt.Println("failed to execute:", err)
			return
		}
		fmt.Println("Result:", output)
	}
}

func Example_serve() {
	// Create a client and serve custom logic (no ADK/LLM required).
	client, err := sdk.NewClient(sdk.Config{
		DataDir: "/tmp/betar-serve",
	})
	if err != nil {
		return
	}
	defer client.Close()

	// Register a custom handler for inbound requests.
	client.Serve(func(ctx context.Context, agentID, input string) (string, error) {
		return "Echo: " + input, nil
	})

	fmt.Println("Serving on peer:", client.PeerID())
}
