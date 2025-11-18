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

type RollD20AdvantageOutput struct {
	Rolls []int  `json:"rolls" jsonschema:"two d20 rolls made with advantage"`
	Total int    `json:"total" jsonschema:"the higher of the two rolls"`
	Note  string `json:"note" jsonschema:"indicates advantage roll"`
}

func RollD20Advantage(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (
	*mcp.CallToolResult,
	RollD20AdvantageOutput,
	error,
) {
	r1 := rand.Intn(20) + 1
	r2 := rand.Intn(20) + 1
	total := max(r1, r2)

	return nil, RollD20AdvantageOutput{
		Rolls: []int{r1, r2},
		Total: total,
		Note:  "Rolled with advantage (kept highest)",
	}, nil
}

type RollD20DisadvantageOutput struct {
	Rolls []int  `json:"rolls" jsonschema:"two d20 rolls made with disadvantage"`
	Total int    `json:"total" jsonschema:"the lower of the two rolls"`
	Note  string `json:"note" jsonschema:"indicates disadvantage roll"`
}

func RollD20Disadvantage(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (
	*mcp.CallToolResult,
	RollD20DisadvantageOutput,
	error,
) {
	r1 := rand.Intn(20) + 1
	r2 := rand.Intn(20) + 1
	total := min(r1, r2)

	return nil, RollD20DisadvantageOutput{
		Rolls: []int{r1, r2},
		Total: total,
		Note:  "Rolled with disadvantage (kept lowest)",
	}, nil
}
