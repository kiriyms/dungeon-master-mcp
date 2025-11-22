package prompts

import (
	"context"
	"fmt"

	"github.com/kiriyms/dungeon-master-mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterCombatPrompts adds all DM assistance prompts to the server
func RegisterCombatPrompts(server *mcp.Server) {
	// Prompt 1: Resolve saving throw with legendary resistance
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "resolve_save_legendary",
			Description: "Guide DM through resolving a saving throw with legendary resistance option",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "monster_id",
					Description: "ID of the monster making the save",
					Required:    true,
				},
				{
					Name:        "save_type",
					Description: "Type of save (STR, DEX, CON, INT, WIS, CHA)",
					Required:    true,
				},
				{
					Name:        "dc",
					Description: "Difficulty class for the save",
					Required:    true,
				},
			},
		},
		handleResolveSavePrompt,
	)

	// Prompt 2: Legendary action decision
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "legendary_action_decision",
			Description: "Suggest optimal legendary action usage based on combat state",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "monster_id",
					Description: "ID of the monster with legendary actions",
					Required:    true,
				},
				{
					Name:        "tactical_context",
					Description: "Current tactical situation (enemy positions, HP, etc)",
					Required:    false,
				},
			},
		},
		handleLegendaryActionPrompt,
	)

	// Prompt 3: Damage application with resistances
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "apply_damage_complex",
			Description: "Calculate and apply damage with resistances, vulnerabilities, and immunities",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "target_id",
					Description: "ID of the target taking damage",
					Required:    true,
				},
				{
					Name:        "damage_amount",
					Description: "Base damage amount",
					Required:    true,
				},
				{
					Name:        "damage_type",
					Description: "Type of damage (fire, cold, slashing, etc)",
					Required:    true,
				},
			},
		},
		handleApplyDamagePrompt,
	)

	// Prompt 4: Turn transition management
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "turn_transition",
			Description: "Manage turn transition with all start-of-turn effects and status summary",
			Arguments:   []*mcp.PromptArgument{},
		},
		handleTurnTransitionPrompt,
	)

	// Prompt 5: Tactical action recommendation
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "tactical_recommendation",
			Description: "Recommend optimal action for a monster based on combat state",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "monster_id",
					Description: "ID of the monster needing action recommendation",
					Required:    true,
				},
				{
					Name:        "available_actions",
					Description: "List of available actions (comma-separated)",
					Required:    false,
				},
			},
		},
		handleTacticalRecommendationPrompt,
	)

	// Prompt 6: Multi-target ability resolution
	server.AddPrompt(
		&mcp.Prompt{
			Name:        "multi_target_ability",
			Description: "Resolve area-of-effect or multi-target abilities efficiently",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "ability_name",
					Description: "Name of the ability being used",
					Required:    true,
				},
				{
					Name:        "target_ids",
					Description: "Comma-separated list of target entity IDs",
					Required:    true,
				},
				{
					Name:        "save_dc",
					Description: "DC for saving throw if applicable",
					Required:    false,
				},
			},
		},
		handleMultiTargetAbilityPrompt,
	)
}

// handleResolveSavePrompt generates instructions for resolving saves with legendary resistance
func handleResolveSavePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	monsterID := req.Params.Arguments["monster_id"]
	saveType := req.Params.Arguments["save_type"]
	dc := req.Params.Arguments["dc"]

	// Fetch monster info from combat state
	cs := tools.GetCombatState()
	if cs == nil {
		return nil, fmt.Errorf("combat state not initialized")
	}
	monster := cs.Entities[monsterID]
	if monster == nil {
		return nil, fmt.Errorf("monster not found: %s", monsterID)
	}

	// Build prompt content based on monster's legendary resistances
	content := fmt.Sprintf(`Resolving Saving Throw for %s

Monster: %s (ID: %s)
Save Type: %s
DC: %s
Current HP: %d/%d
Legendary Resistances Remaining: %d

Process:
1. Roll d20 for the %s save
2. Add the monster's save bonus (check monster stat block)
3. Compare total to DC %s
4. If the save fails AND the monster has legendary resistances remaining, ask the DM:
   "The save failed. Would you like to use one of the %d remaining legendary resistances to automatically succeed?"
5. If legendary resistance is used:
   - Decrement legendary resistances by 1
   - Treat the save as a success
   - Apply no effect from the spell/ability
6. If legendary resistance is not used or unavailable:
   - Apply the full effect of the failed save

Use the make_saving_throw tool to handle the dice roll and legendary resistance decision automatically.`,
		monster.Name,
		monster.Name,
		monsterID,
		saveType,
		dc,
		monster.CurrentHP,
		monster.MaxHP,
		monster.LegendaryResistances,
		saveType,
		dc,
		monster.LegendaryResistances,
	)

	return &mcp.GetPromptResult{
		Description: "Instructions for resolving a save with legendary resistance",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}

// handleLegendaryActionPrompt suggests optimal legendary action usage
func handleLegendaryActionPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	monsterID := req.Params.Arguments["monster_id"]
	tacticalContext := req.Params.Arguments["tactical_context"]

	cs := tools.GetCombatState()
	if cs == nil {
		return nil, fmt.Errorf("combat state not initialized")
	}

	monster := cs.Entities[monsterID]
	if monster == nil {
		return nil, fmt.Errorf("monster not found: %s", monsterID)
	}

	// Build tactical analysis
	content := fmt.Sprintf(`Legendary Action Decision for %s

Monster Status:
- Current HP: %d/%d (%.0f%% remaining)
- Legendary Actions Available: %d/%d
- Position in Initiative: Turn %d of %d, Round %d

Tactical Context: %s

Available Legendary Actions:
`,
		monster.Name,
		monster.CurrentHP,
		monster.MaxHP,
		float64(monster.CurrentHP)/float64(monster.MaxHP)*100,
		monster.LegendaryActions,
		monster.MaxLegendaryActions,
		cs.CurrentTurn+1,
		len(cs.TurnOrder),
		cs.RoundNumber,
		tacticalContext,
	)

	// Query monster resources for legendary action options
	// In production, this would read from the monster stat block resource
	content += `
1. Detect (Cost 1) - Make a Wisdom (Perception) check
   When to use: When tracking hidden enemies or searching for threats

2. Tail Attack (Cost 1) - Make a single melee attack
   When to use: Against nearby targets, consistent damage output

3. Wing Attack (Cost 2) - AoE knockdown effect
   When to use: When surrounded by multiple melee attackers, creates distance

Recommendation Process:
1. Assess immediate threats (how many enemies are in melee range?)
2. Check HP status (below 50%? Prioritize defensive options)
3. Consider action economy (save actions for key moments vs use them consistently)
4. Evaluate the monster's position in initiative (how soon until their next turn?)

Current Recommendation:
`
	// Add tactical recommendation based on HP and situation
	if float64(monster.CurrentHP)/float64(monster.MaxHP) < 0.5 {
		content += `The monster is below 50% HP. Consider defensive legendary actions:
- Wing Attack (2 actions) to create distance if surrounded
- Save actions if the monster's turn is coming soon`
	} else {
		content += `The monster is in good health. Consider aggressive legendary actions:
- Tail Attack (1 action) for consistent damage
- Wing Attack (2 actions) if multiple enemies are clustered`
	}

	content += fmt.Sprintf(`

Use the use_legendary_action tool with:
- monster_id: "%s"
- action_name: [chosen action name]
- cost: [action point cost]`, monsterID)

	return &mcp.GetPromptResult{
		Description: "Tactical recommendation for legendary action usage",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}

// handleApplyDamagePrompt creates a detailed damage calculation prompt
func handleApplyDamagePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	targetID := req.Params.Arguments["target_id"]
	damageAmount := req.Params.Arguments["damage_amount"]
	damageType := req.Params.Arguments["damage_type"]

	cs := tools.GetCombatState()
	if cs == nil {
		return nil, fmt.Errorf("combat state not initialized")
	}

	target := cs.Entities[targetID]
	if target == nil {
		return nil, fmt.Errorf("target not found: %s", targetID)
	}

	content := fmt.Sprintf(`Damage Application for %s

Target: %s (ID: %s)
Base Damage: %s
Damage Type: %s
Current HP: %d/%d
AC: %d

Damage Calculation Process:
1. Start with base damage: %s
2. Check target's damage resistances, vulnerabilities, and immunities (see monster stat block)
3. Apply modifiers:
   - Resistance: damage halved (rounded down)
   - Vulnerability: damage doubled
   - Immunity: no damage taken
4. Subtract final damage from current HP
5. Check for status changes:
   - HP = 0: creature becomes unconscious or dies
   - HP <= MaxHP/2: creature is bloodied (informational)

For monster stat blocks, query the resource: monster://stat_block/%s

Use the apply_damage tool with:
- target_id: "%s"
- damage: %s
- damage_type: "%s"`,
		target.Name,
		target.Name,
		targetID,
		damageAmount,
		damageType,
		target.CurrentHP,
		target.MaxHP,
		target.AC,
		damageAmount,
		target.MonsterName,
		targetID,
		damageAmount,
		damageType,
	)

	return &mcp.GetPromptResult{
		Description: "Detailed damage calculation with resistance handling",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}

// handleTurnTransitionPrompt manages turn advancement
func handleTurnTransitionPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	cs := tools.GetCombatState()
	if cs == nil {
		return nil, fmt.Errorf("combat state not initialized")
	}

	nextTurn := (cs.CurrentTurn + 1) % len(cs.TurnOrder)
	nextRound := cs.RoundNumber
	if nextTurn == 0 {
		nextRound++
	}

	nextEntityID := cs.TurnOrder[nextTurn]
	nextEntity := cs.Entities[nextEntityID]

	content := fmt.Sprintf(`Turn Transition - Round %d

Current State:
- Current Turn: %d/%d
- Next Entity: %s (ID: %s)
- Next Entity HP: %d/%d

Turn Transition Checklist:
1. End current turn effects:
   - Check for end-of-turn condition expirations
   - Process any ongoing damage or healing

2. Advance to next turn using next_turn tool

3. Start-of-turn effects for %s:
   - Reset legendary actions (if applicable)
   - Decrement condition durations
   - Apply regeneration (if applicable)
   - Check for start-of-turn triggers

4. Present combat status:
   - Current entity's available actions
   - Remaining resources (legendary actions, abilities)
   - Active conditions
   - Nearby threats and allies

Use the next_turn tool to handle this automatically.

Combat Status Summary:
`,
		nextRound,
		cs.CurrentTurn+1,
		len(cs.TurnOrder),
		nextEntity.Name,
		nextEntityID,
		nextEntity.CurrentHP,
		nextEntity.MaxHP,
		nextEntity.Name,
	)

	// Add status for all combatants
	for _, id := range cs.TurnOrder {
		entity := cs.Entities[id]
		conditions := ""
		if len(entity.Conditions) > 0 {
			condList := []string{}
			for cond := range entity.Conditions {
				condList = append(condList, cond)
			}
			conditions = fmt.Sprintf(" [%v]", condList)
		}
		content += fmt.Sprintf("- %s: %d/%d HP%s\n", entity.Name, entity.CurrentHP, entity.MaxHP, conditions)
	}

	return &mcp.GetPromptResult{
		Description: "Complete turn transition management",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}

// handleTacticalRecommendationPrompt provides action recommendations
func handleTacticalRecommendationPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	monsterID := req.Params.Arguments["monster_id"]
	availableActions := req.Params.Arguments["available_actions"]

	cs := tools.GetCombatState()
	if cs == nil {
		return nil, fmt.Errorf("combat state not initialized")
	}

	monster := cs.Entities[monsterID]
	if monster == nil {
		return nil, fmt.Errorf("monster not found: %s", monsterID)
	}

	content := fmt.Sprintf(`Tactical Action Recommendation for %s

Monster Status:
- Current HP: %d/%d (%.0f%% remaining)
- Available Actions: %s
- Position: Round %d, Turn %d
- Active Conditions: %v

Tactical Analysis Framework:
1. Threat Assessment:
   - Count nearby enemy combatants
   - Identify highest-threat targets (low HP, spellcasters)
   - Assess area control opportunities

2. Resource Management:
   - Check for recharge abilities (roll if applicable)
   - Consider limited-use abilities vs at-will actions
   - Save powerful abilities for strategic moments

3. Action Priority:
   - High HP (>75%%): Aggressive actions, use powerful abilities
   - Medium HP (25-75%%): Balanced approach, damage + positioning
   - Low HP (<25%%): Defensive actions, escape or area denial

4. Common Action Recommendations:
   - Multiattack: Standard choice for consistent damage
   - Breath Weapon/Special Ability: Use when recharged and multiple targets available
   - Legendary Actions: Consider saving for reactions to PC actions
   - Movement: Reposition if focused by multiple enemies

Current Recommendation:
`,
		monster.Name,
		monster.CurrentHP,
		monster.MaxHP,
		float64(monster.CurrentHP)/float64(monster.MaxHP)*100,
		availableActions,
		cs.RoundNumber,
		cs.CurrentTurn+1,
		monster.Conditions,
	)

	// Tactical recommendation based on HP percentage
	hpPercent := float64(monster.CurrentHP) / float64(monster.MaxHP)
	if hpPercent > 0.75 {
		content += `The monster is at high HP. Recommended approach:
- Use powerful, limited-use abilities if appropriate targets are available
- Focus on highest-threat targets (spellcasters, low-HP strikers)
- Utilize multiattack for consistent damage output
- Consider area-of-effect abilities if enemies are clustered`
	} else if hpPercent > 0.25 {
		content += `The monster is at medium HP. Recommended approach:
- Balance offense and defense
- Continue with reliable attack actions (multiattack)
- Save legendary resistances for critical saves
- Consider tactical repositioning if heavily outnumbered`
	} else {
		content += `The monster is at low HP. Recommended approach:
- Prioritize survival (use legendary resistances aggressively)
- Consider defensive legendary actions (Wing Attack for distance)
- Use area denial or escape tactics
- Focus on high-value targets if going down is inevitable`
	}

	content += fmt.Sprintf(`

To query full action details, use resource: monster://stat_block/%s`, monster.MonsterName)

	return &mcp.GetPromptResult{
		Description: "Tactical action recommendation based on combat state",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}

// handleMultiTargetAbilityPrompt helps resolve area effects
func handleMultiTargetAbilityPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	abilityName := req.Params.Arguments["ability_name"]
	targetIDs := req.Params.Arguments["target_ids"]
	saveDC := req.Params.Arguments["save_dc"]

	content := fmt.Sprintf(`Multi-Target Ability Resolution: %s

Targets: %s
Save DC: %s

Efficient Resolution Process:
1. For each target in the list:
   a. Roll saving throw using make_saving_throw tool
   b. Record result (success/failure)
   c. Note if legendary resistance was used

2. Apply effects based on save results:
   - Failed saves: full effect
   - Successful saves: usually half damage or no effect
   - Check ability description for specific outcomes

3. Apply damage to all targets using apply_damage tool:
   - Failed saves: full damage
   - Successful saves: half damage (if applicable)

4. Update combat status using next_turn tool if this ends the current action

Batch Processing Template:
`,
		abilityName,
		targetIDs,
		saveDC,
	)

	// Add individual target sections
	content += `
For each target:
  make_saving_throw(entity_id: [target_id], save_type: [save_type], dc: [dc])
  IF save failed:
    apply_damage(target_id: [target_id], damage: [full_damage], damage_type: [type])
  ELSE:
    apply_damage(target_id: [target_id], damage: [half_damage], damage_type: [type])

This batch approach minimizes tool calls while maintaining accuracy.`

	return &mcp.GetPromptResult{
		Description: "Streamlined multi-target ability resolution",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}
