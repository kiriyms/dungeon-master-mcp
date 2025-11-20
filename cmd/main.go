package main

import (
	"context"
	"log"

	"github.com/kiriyms/dungeon-master-mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Create the MCP server with implementation metadata
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "dungeon-master-mcp",
			Version: "1.0.0",
		},
		nil, // Use default server options
	)

	// Register all combat management tools
	// These handle initiative tracking, damage, healing, conditions, etc.
	tools.RegisterCombatTools(server)
	log.Println("Registered Tools: combat management, damage calculation, legendary actions")

	log.Println("D&D Combat MCP Server starting...")
	
	// Run the server over stdin/stdout for Claude integration
	// This allows the MCP client (like Claude Desktop) to communicate with the server
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
