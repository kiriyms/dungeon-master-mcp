package tools

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SimpleRollOutput struct {
	Roll int    `json:"roll" jsonschema:"result of rolling 1d20"`
	Note string `json:"note" jsonschema:"a human-readable message about the roll"`
}

func RollD20(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (
	*mcp.CallToolResult,
	SimpleRollOutput,
	error,
) {
	roll := rand.Intn(20) + 1
	output := SimpleRollOutput{
		Roll: roll,
		Note: fmt.Sprintf("Rolled a %d on a d20", roll),
	}
	return nil, output, nil
}
