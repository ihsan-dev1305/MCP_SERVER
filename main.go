package main

import (
	"context"
	"log"

	"dummyjson-explorer/api"
	"dummyjson-explorer/mcpserver"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Create the DummyJSON API client.
	client := api.NewClient()

	// Create the MCP server.
	server := mcpserver.NewServer(client)

	// Start the server using standard input/output.
	if err := server.Run(
		context.Background(),
		&mcp.StdioTransport{},
	); err != nil {
		log.Fatal(err)
	}
}
