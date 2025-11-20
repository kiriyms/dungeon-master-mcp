package tools

import (
	"context"
	"fmt"
	"math/rand"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CombatState tracks the current combat session
type CombatState struct {
	Entities    map[string]*Entity // entity_id -> Entity
	TurnOrder   []string           // ordered list of entity IDs
	CurrentTurn int                // index in TurnOrder
	RoundNumber int
}

// Entity represents a combatant (PC or monster)
type Entity struct {
	ID                   string
	Name                 string
	InitiativeRoll       int
	MaxHP                int
	CurrentHP            int
	AC                   int
	Conditions           map[string]int // condition -> turns remaining (-1 = permanent)
	Resources            map[string]int // resource_name -> current count
	IsMonster            bool
	MonsterName          string // for loading stats
	LegendaryActions     int    // remaining this round
	MaxLegendaryActions  int
	LegendaryResistances int
}

var combatState *CombatState

// RegisterCombatTools adds all combat-related tools to the server
func RegisterCombatTools(server *mcp.Server) {
	// Initialize combat state
	combatState = &CombatState{
		Entities:  make(map[string]*Entity),
		TurnOrder: []string{},
	}

	// Tool 1: Start Combat
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "start_combat",
			Description: "Initialize combat with party members and monsters, assigns initiative",
		},
		handleStartCombat,
	)

	// Tool 2: Next Turn
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "next_turn",
			Description: "Advance to next turn, handle start-of-turn effects",
		},
		handleNextTurn,
	)

	// Tool 3: Apply Damage
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "apply_damage",
			Description: "Apply damage to target with resistance/vulnerability/immunity calculations",
		},
		handleApplyDamage,
	)

	// Tool 4: Apply Healing
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "apply_healing",
			Description: "Heal target and update hit points",
		},
		handleApplyHealing,
	)

	// Tool 5: Add Condition
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "add_condition",
			Description: "Apply a condition to an entity with duration",
		},
		handleAddCondition,
	)

	// Tool 6: Make Saving Throw
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "make_saving_throw",
			Description: "Roll saving throw with legendary resistance option",
		},
		handleSavingThrow,
	)

	// Tool 7: Use Legendary Action
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "use_legendary_action",
			Description: "Use a monster's legendary action",
		},
		handleLegendaryAction,
	)

	// Tool 8: Track Resource
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "track_resource",
			Description: "Track a resource like spell slots or limited abilities",
		},
		handleTrackResource,
	)
}

// StartCombatInput defines the structure for starting combat
type StartCombatInput struct {
	Entities []EntityInit `json:"entities" jsonschema:"List of combatants with initiative"`
}

type EntityInit struct {
	ID          string `json:"id" jsonschema:"Unique identifier"`
	Name        string `json:"name" jsonschema:"Display name"`
	Initiative  int    `json:"initiative" jsonschema:"Initiative roll"`
	HP          int    `json:"hp" jsonschema:"Max hit points"`
	AC          int    `json:"ac" jsonschema:"Armor class"`
	IsMonster   bool   `json:"is_monster" jsonschema:"Whether this is a monster"`
	MonsterName string `json:"monster_name,omitempty" jsonschema:"Monster type name for loading stats"`
}

type StartCombatOutput struct {
	TurnOrder []string `json:"turn_order" jsonschema:"Initiative order by entity ID"`
	Message   string   `json:"message" jsonschema:"Status message"`
}

func handleStartCombat(ctx context.Context, req *mcp.CallToolRequest, input StartCombatInput) (*mcp.CallToolResult, StartCombatOutput, error) {
	// Reset combat state
	combatState.Entities = make(map[string]*Entity)
	combatState.TurnOrder = []string{}
	combatState.CurrentTurn = 0
	combatState.RoundNumber = 1

	// Create entities
	for _, e := range input.Entities {
		entity := &Entity{
			ID:             e.ID,
			Name:           e.Name,
			InitiativeRoll: e.Initiative,
			MaxHP:          e.HP,
			CurrentHP:      e.HP,
			AC:             e.AC,
			Conditions:     make(map[string]int),
			Resources:      make(map[string]int),
			IsMonster:      e.IsMonster,
			MonsterName:    e.MonsterName,
		}

		// Load monster stats if applicable
		if e.IsMonster && e.MonsterName != "" {
			loadMonsterStats(entity)
		}

		combatState.Entities[e.ID] = entity
	}

	// Sort by initiative (descending)
	type initPair struct {
		id   string
		init int
	}
	pairs := []initPair{}
	for id, e := range combatState.Entities {
		pairs = append(pairs, initPair{id, e.InitiativeRoll})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].init > pairs[j].init
	})

	for _, p := range pairs {
		combatState.TurnOrder = append(combatState.TurnOrder, p.id)
	}

	return nil, StartCombatOutput{
		TurnOrder: combatState.TurnOrder,
		Message:   fmt.Sprintf("Combat started with %d combatants. Round 1, turn 1.", len(combatState.Entities)),
	}, nil
}

// NextTurnInput defines advancing the turn
type NextTurnInput struct{}

type NextTurnOutput struct {
	CurrentEntityID   string            `json:"current_entity_id"`
	CurrentEntityName string            `json:"current_entity_name"`
	RoundNumber       int               `json:"round_number"`
	Effects           []string          `json:"effects" jsonschema:"Start of turn effects applied"`
	CombatStatus      map[string]string `json:"combat_status" jsonschema:"HP and conditions summary"`
}

func handleNextTurn(ctx context.Context, req *mcp.CallToolRequest, input NextTurnInput) (*mcp.CallToolResult, NextTurnOutput, error) {
	// Advance turn
	combatState.CurrentTurn++
	if combatState.CurrentTurn >= len(combatState.TurnOrder) {
		combatState.CurrentTurn = 0
		combatState.RoundNumber++
	}

	currentID := combatState.TurnOrder[combatState.CurrentTurn]
	current := combatState.Entities[currentID]

	effects := []string{}

	// Reset legendary actions at start of monster turn
	if current.IsMonster && current.MaxLegendaryActions > 0 {
		current.LegendaryActions = current.MaxLegendaryActions
		effects = append(effects, fmt.Sprintf("Legendary actions reset to %d", current.MaxLegendaryActions))
	}

	// Process conditions (decrement duration)
	for condition, duration := range current.Conditions {
		if duration > 0 {
			current.Conditions[condition]--
			if current.Conditions[condition] == 0 {
				delete(current.Conditions, condition)
				effects = append(effects, fmt.Sprintf("Condition '%s' ended", condition))
			}
		}
	}

	// Build status summary
	status := make(map[string]string)
	for id, e := range combatState.Entities {
		condList := []string{}
		for c := range e.Conditions {
			condList = append(condList, c)
		}
		condStr := ""
		if len(condList) > 0 {
			condStr = fmt.Sprintf(" [%v]", condList)
		}
		status[id] = fmt.Sprintf("%s: %d/%d HP%s", e.Name, e.CurrentHP, e.MaxHP, condStr)
	}

	return nil, NextTurnOutput{
		CurrentEntityID:   currentID,
		CurrentEntityName: current.Name,
		RoundNumber:       combatState.RoundNumber,
		Effects:           effects,
		CombatStatus:      status,
	}, nil
}

// ApplyDamageInput defines damage application
type ApplyDamageInput struct {
	TargetID   string `json:"target_id" jsonschema:"Entity receiving damage"`
	Damage     int    `json:"damage" jsonschema:"Damage amount"`
	DamageType string `json:"damage_type" jsonschema:"Type of damage (fire, slashing, etc)"`
}

type ApplyDamageOutput struct {
	FinalDamage   int    `json:"final_damage"`
	RemainingHP   int    `json:"remaining_hp"`
	Message       string `json:"message"`
	IsUnconscious bool   `json:"is_unconscious"`
}

func handleApplyDamage(ctx context.Context, req *mcp.CallToolRequest, input ApplyDamageInput) (*mcp.CallToolResult, ApplyDamageOutput, error) {
	target := combatState.Entities[input.TargetID]
	if target == nil {
		return nil, ApplyDamageOutput{}, fmt.Errorf("target not found: %s", input.TargetID)
	}

	// Apply resistance/vulnerability/immunity (simplified - would normally check monster stats)
	finalDamage := input.Damage
	modifier := ""

	// Check resistances from Resources
	if _, ok := target.Resources["resistances"]; ok {
		// In real implementation, parse resistance types
		finalDamage = input.Damage / 2
		modifier = " (resisted)"
	}

	target.CurrentHP -= finalDamage
	if target.CurrentHP < 0 {
		target.CurrentHP = 0
	}

	isUnconscious := target.CurrentHP == 0

	return nil, ApplyDamageOutput{
		FinalDamage:   finalDamage,
		RemainingHP:   target.CurrentHP,
		Message:       fmt.Sprintf("%s takes %d %s damage%s. %d HP remaining.", target.Name, finalDamage, input.DamageType, modifier, target.CurrentHP),
		IsUnconscious: isUnconscious,
	}, nil
}

// ApplyHealingInput defines healing
type ApplyHealingInput struct {
	TargetID string `json:"target_id"`
	Amount   int    `json:"amount"`
}

type ApplyHealingOutput struct {
	AmountHealed int    `json:"amount_healed"`
	CurrentHP    int    `json:"current_hp"`
	Message      string `json:"message"`
}

func handleApplyHealing(ctx context.Context, req *mcp.CallToolRequest, input ApplyHealingInput) (*mcp.CallToolResult, ApplyHealingOutput, error) {
	target := combatState.Entities[input.TargetID]
	if target == nil {
		return nil, ApplyHealingOutput{}, fmt.Errorf("target not found: %s", input.TargetID)
	}

	before := target.CurrentHP
	target.CurrentHP += input.Amount
	if target.CurrentHP > target.MaxHP {
		target.CurrentHP = target.MaxHP
	}
	healed := target.CurrentHP - before

	return nil, ApplyHealingOutput{
		AmountHealed: healed,
		CurrentHP:    target.CurrentHP,
		Message:      fmt.Sprintf("%s healed for %d HP. Now at %d/%d.", target.Name, healed, target.CurrentHP, target.MaxHP),
	}, nil
}

// AddConditionInput defines adding conditions
type AddConditionInput struct {
	TargetID  string `json:"target_id"`
	Condition string `json:"condition" jsonschema:"Condition name (stunned, prone, etc)"`
	Duration  int    `json:"duration" jsonschema:"Turns remaining, -1 for permanent"`
}

type AddConditionOutput struct {
	Message string `json:"message"`
}

func handleAddCondition(ctx context.Context, req *mcp.CallToolRequest, input AddConditionInput) (*mcp.CallToolResult, AddConditionOutput, error) {
	target := combatState.Entities[input.TargetID]
	if target == nil {
		return nil, AddConditionOutput{}, fmt.Errorf("target not found: %s", input.TargetID)
	}

	target.Conditions[input.Condition] = input.Duration
	durationMsg := fmt.Sprintf("%d turns", input.Duration)
	if input.Duration == -1 {
		durationMsg = "permanent"
	}

	return nil, AddConditionOutput{
		Message: fmt.Sprintf("%s is now %s (%s).", target.Name, input.Condition, durationMsg),
	}, nil
}

// SavingThrowInput defines saving throws
type SavingThrowInput struct {
	EntityID string `json:"entity_id"`
	SaveType string `json:"save_type" jsonschema:"STR, DEX, CON, INT, WIS, CHA"`
	DC       int    `json:"dc" jsonschema:"Difficulty class"`
}

type SavingThrowOutput struct {
	Roll                      int    `json:"roll"`
	Bonus                     int    `json:"bonus"`
	Total                     int    `json:"total"`
	Success                   bool   `json:"success"`
	UsedLegendaryResistance   bool   `json:"used_legendary_resistance"`
	RemainingLegendaryResists int    `json:"remaining_legendary_resists"`
	Message                   string `json:"message"`
}

func handleSavingThrow(ctx context.Context, req *mcp.CallToolRequest, input SavingThrowInput) (*mcp.CallToolResult, SavingThrowOutput, error) {
	entity := combatState.Entities[input.EntityID]
	if entity == nil {
		return nil, SavingThrowOutput{}, fmt.Errorf("entity not found: %s", input.EntityID)
	}

	// Roll d20
	roll := rand.Intn(20) + 1

	// Get save bonus (simplified - would normally check monster stats)
	bonus := 0
	if entity.IsMonster {
		bonus = 3 // placeholder
	}

	total := roll + bonus
	success := total >= input.DC

	usedLegendary := false
	if !success && entity.LegendaryResistances > 0 {
		// Auto-succeed using legendary resistance
		success = true
		usedLegendary = true
		entity.LegendaryResistances--
	}

	message := fmt.Sprintf("%s rolled %d+%d=%d vs DC %d: %s",
		entity.Name, roll, bonus, total, input.DC,
		map[bool]string{true: "SUCCESS", false: "FAILURE"}[success])

	if usedLegendary {
		message += fmt.Sprintf(" (used legendary resistance, %d remaining)", entity.LegendaryResistances)
	}

	return nil, SavingThrowOutput{
		Roll:                      roll,
		Bonus:                     bonus,
		Total:                     total,
		Success:                   success,
		UsedLegendaryResistance:   usedLegendary,
		RemainingLegendaryResists: entity.LegendaryResistances,
		Message:                   message,
	}, nil
}

// LegendaryActionInput defines using legendary actions
type LegendaryActionInput struct {
	MonsterID  string `json:"monster_id"`
	ActionName string `json:"action_name"`
	Cost       int    `json:"cost" jsonschema:"Number of legendary actions to spend"`
}

type LegendaryActionOutput struct {
	Success          bool   `json:"success"`
	RemainingActions int    `json:"remaining_actions"`
	Message          string `json:"message"`
}

func handleLegendaryAction(ctx context.Context, req *mcp.CallToolRequest, input LegendaryActionInput) (*mcp.CallToolResult, LegendaryActionOutput, error) {
	monster := combatState.Entities[input.MonsterID]
	if monster == nil {
		return nil, LegendaryActionOutput{}, fmt.Errorf("monster not found: %s", input.MonsterID)
	}

	if monster.LegendaryActions < input.Cost {
		return nil, LegendaryActionOutput{
			Success:          false,
			RemainingActions: monster.LegendaryActions,
			Message:          fmt.Sprintf("Insufficient legendary actions. Has %d, needs %d.", monster.LegendaryActions, input.Cost),
		}, nil
	}

	monster.LegendaryActions -= input.Cost

	return nil, LegendaryActionOutput{
		Success:          true,
		RemainingActions: monster.LegendaryActions,
		Message:          fmt.Sprintf("%s uses %s (cost %d). %d legendary actions remaining.", monster.Name, input.ActionName, input.Cost, monster.LegendaryActions),
	}, nil
}

// TrackResourceInput defines resource tracking
type TrackResourceInput struct {
	EntityID     string `json:"entity_id"`
	ResourceName string `json:"resource_name"`
	CurrentValue int    `json:"current_value"`
}

type TrackResourceOutput struct {
	Message string `json:"message"`
}

func handleTrackResource(ctx context.Context, req *mcp.CallToolRequest, input TrackResourceInput) (*mcp.CallToolResult, TrackResourceOutput, error) {
	entity := combatState.Entities[input.EntityID]
	if entity == nil {
		return nil, TrackResourceOutput{}, fmt.Errorf("entity not found: %s", input.EntityID)
	}

	entity.Resources[input.ResourceName] = input.CurrentValue

	return nil, TrackResourceOutput{
		Message: fmt.Sprintf("%s now has %d %s.", entity.Name, input.CurrentValue, input.ResourceName),
	}, nil
}

// loadMonsterStats populates monster-specific stats from Resources
func loadMonsterStats(entity *Entity) {
	// This would normally query the Resources for monster stat blocks
	// For now, set some defaults
	if entity.MonsterName == "Ancient Red Dragon" {
		entity.MaxLegendaryActions = 3
		entity.LegendaryActions = 3
		entity.LegendaryResistances = 3
	}
}
