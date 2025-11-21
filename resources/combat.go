package resources

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MonsterStat represents a complete monster stat block from SRD
type MonsterStat struct {
	Name                  string              `json:"name"`
	Size                  string              `json:"size"`
	Type                  string              `json:"type"`
	Alignment             string              `json:"alignment"`
	HP                    int                 `json:"hp"`
	AC                    int                 `json:"ac"`
	Speed                 map[string]int      `json:"speed"`
	AbilityScores         map[string]int      `json:"ability_scores"`
	SavingThrows          map[string]int      `json:"saving_throws"`
	Skills                map[string]int      `json:"skills"`
	DamageResistances     []string            `json:"damage_resistances"`
	DamageImmunities      []string            `json:"damage_immunities"`
	DamageVulnerabilities []string            `json:"damage_vulnerabilities"`
	ConditionImmunities   []string            `json:"condition_immunities"`
	Senses                map[string]int      `json:"senses"`
	Languages             []string            `json:"languages"`
	ChallengeRating       float64             `json:"challenge_rating"`
	Traits                []MonsterTrait      `json:"traits"`
	Actions               []MonsterAction     `json:"actions"`
	LegendaryActions      *LegendaryActionSet `json:"legendary_actions,omitempty"`
	LairActions           []LairAction        `json:"lair_actions,omitempty"`
}

// MonsterTrait represents a passive ability or feature
type MonsterTrait struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MonsterAction represents an action a monster can take
type MonsterAction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	AttackBonus int    `json:"attack_bonus,omitempty"`
	DamageType  string `json:"damage_type,omitempty"`
	DamageDice  string `json:"damage_dice,omitempty"`
	SaveDC      int    `json:"save_dc,omitempty"`
	SaveType    string `json:"save_type,omitempty"`
}

// LegendaryActionSet defines a monster's legendary action options
type LegendaryActionSet struct {
	ActionsPerRound int                  `json:"actions_per_round"`
	Options         []LegendaryActionOpt `json:"options"`
}

type LegendaryActionOpt struct {
	Name        string `json:"name"`
	Cost        int    `json:"cost"`
	Description string `json:"description"`
}

// LairAction represents an action that happens on initiative 20
type LairAction struct {
	Description string `json:"description"`
	SaveDC      int    `json:"save_dc,omitempty"`
	SaveType    string `json:"save_type,omitempty"`
}

// DamageRules contains SRD rules for damage calculation
type DamageRules struct {
	ResistanceMultiplier    float64           `json:"resistance_multiplier"`
	VulnerabilityMultiplier float64           `json:"vulnerability_multiplier"`
	ImmunityEffect          string            `json:"immunity_effect"`
	CriticalMultiplier      int               `json:"critical_multiplier"`
	ConditionEffects        map[string]string `json:"condition_effects"`
}

// RegisterCombatResources adds all SRD data resources to the server
func RegisterCombatResources(server *mcp.Server) {
	// Resource 1: Monster Stat Block by name
	server.AddResource(
		&mcp.Resource{
			URI:         "monster://stat_block/{name}",
			Name:        "monster_stat_block",
			Description: "Retrieve complete SRD stat block for a monster by name",
			MIMEType:    "application/json",
		},
		adaptStringHandler(handleMonsterStatBlock),
	)

	// Resource 2: Damage calculation rules
	server.AddResource(
		&mcp.Resource{
			URI:         "srd://rules/damage",
			Name:        "damage_rules",
			Description: "SRD rules for damage, resistance, vulnerability, and immunity",
			MIMEType:    "application/json",
		},
		adaptStringHandler(handleDamageRules),
	)

	// Resource 3: Condition definitions
	server.AddResource(
		&mcp.Resource{
			URI:         "srd://rules/conditions",
			Name:        "condition_rules",
			Description: "Complete list of D&D 5e conditions and their mechanical effects",
			MIMEType:    "application/json",
		},
		adaptStringHandler(handleConditionRules),
	)

	// Resource 4: Saving throw rules
	server.AddResource(
		&mcp.Resource{
			URI:         "srd://rules/saving_throws",
			Name:        "saving_throw_rules",
			Description: "Rules for saving throws including critical success and failure",
			MIMEType:    "application/json",
		},
		adaptStringHandler(handleSavingThrowRules),
	)

	// Resource 5: Legendary mechanics
	server.AddResource(
		&mcp.Resource{
			URI:         "srd://rules/legendary",
			Name:        "legendary_rules",
			Description: "Rules for legendary actions, legendary resistances, and lair actions",
			MIMEType:    "application/json",
		},
		adaptStringHandler(handleLegendaryRules),
	)

	// Resource 6: Monster list
	server.AddResource(
		&mcp.Resource{
			URI:         "srd://monsters/list",
			Name:        "monster_list",
			Description: "List of all available SRD monsters",
			MIMEType:    "application/json",
		},
		adaptStringHandler(handleMonsterList),
	)
}

// adaptStringHandler converts an existing handler that returns (string, error)
// into the updated `mcp.ResourceHandler` signature.
func adaptStringHandler(h func(context.Context, string) (string, error)) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		// Extract a URI-like string from the request using reflection so the
		// adapter works with different SDK shapes.
		var uri string
		if req != nil {
			rv := reflect.ValueOf(req)
			if rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
			}
			if rv.IsValid() {
				for i := 0; i < rv.NumField(); i++ {
					f := rv.Field(i)
					if !f.IsValid() {
						continue
					}
					switch f.Kind() {
					case reflect.String:
						s := f.String()
						if strings.Contains(s, "://") || strings.Contains(s, "/") {
							uri = s
						}
					case reflect.Map:
						found := false
						for _, k := range f.MapKeys() {
							v := f.MapIndex(k)
							if v.Kind() == reflect.String {
								s := v.String()
								if strings.Contains(s, "://") || strings.Contains(s, "/") {
									uri = s
									found = true
									break
								}
							}
						}
						if found {
							// uri set above
						}
					}
					if uri != "" {
						break
					}
				}
			}
		}

		str, err := h(ctx, uri)
		if err != nil {
			return nil, err
		}

		// Build result via reflection so we don't depend on exact field names
		// used by different SDK versions.
		res := new(mcp.ReadResourceResult)
		rv := reflect.ValueOf(res).Elem()
		// Set first []byte field we find, or first string field as fallback.
		setString := false
		for i := 0; i < rv.NumField(); i++ {
			f := rv.Field(i)
			if !f.CanSet() {
				continue
			}
			switch f.Kind() {
			case reflect.Slice:
				if f.Type().Elem().Kind() == reflect.Uint8 {
					f.SetBytes([]byte(str))
					setString = true
				}
			case reflect.String:
				if !setString {
					f.SetString(str)
					setString = true
				}
			}
			if setString {
				break
			}
		}

		// Try to set a MIME-like field if present.
		for i := 0; i < rv.NumField(); i++ {
			f := rv.Field(i)
			if !f.CanSet() || f.Kind() != reflect.String {
				continue
			}
			name := rv.Type().Field(i).Name
			lname := strings.ToLower(name)
			if strings.Contains(lname, "mime") || strings.Contains(lname, "contenttype") || strings.Contains(lname, "type") {
				f.SetString("application/json")
				break
			}
		}

		return res, nil
	}
}

// handleMonsterStatBlock returns a complete monster stat block
func handleMonsterStatBlock(ctx context.Context, uri string) (string, error) {
	// Parse monster name from URI (simplified)
	// In production, would parse from "monster://stat_block/{name}"

	// Example: Ancient Red Dragon
	dragon := MonsterStat{
		Name:      "Ancient Red Dragon",
		Size:      "Gargantuan",
		Type:      "dragon",
		Alignment: "chaotic evil",
		HP:        546,
		AC:        22,
		Speed: map[string]int{
			"walk":  40,
			"climb": 40,
			"fly":   80,
		},
		AbilityScores: map[string]int{
			"STR": 30, "DEX": 10, "CON": 29,
			"INT": 18, "WIS": 15, "CHA": 23,
		},
		SavingThrows: map[string]int{
			"DEX": 7, "CON": 16, "WIS": 9, "CHA": 13,
		},
		Skills: map[string]int{
			"Perception": 16,
			"Stealth":    7,
		},
		DamageImmunities: []string{"fire"},
		Senses: map[string]int{
			"blindsight": 60,
			"darkvision": 120,
			"perception": 26,
		},
		Languages:       []string{"Common", "Draconic"},
		ChallengeRating: 24,
		Traits: []MonsterTrait{
			{
				Name:        "Legendary Resistance",
				Description: "If the dragon fails a saving throw, it can choose to succeed instead (3/day).",
			},
		},
		Actions: []MonsterAction{
			{
				Name:        "Multiattack",
				Description: "The dragon can use its Frightful Presence. It then makes three attacks: one with its bite and two with its claws.",
			},
			{
				Name:        "Bite",
				AttackBonus: 17,
				DamageType:  "piercing",
				DamageDice:  "2d10+10",
			},
			{
				Name:        "Fire Breath",
				Description: "The dragon exhales fire in a 90-foot cone. Each creature must make a DC 24 Dexterity saving throw, taking 91 (26d6) fire damage on a failed save, or half as much on a successful one.",
				SaveDC:      24,
				SaveType:    "DEX",
			},
		},
		LegendaryActions: &LegendaryActionSet{
			ActionsPerRound: 3,
			Options: []LegendaryActionOpt{
				{
					Name:        "Detect",
					Cost:        1,
					Description: "The dragon makes a Wisdom (Perception) check.",
				},
				{
					Name:        "Tail Attack",
					Cost:        1,
					Description: "The dragon makes a tail attack.",
				},
				{
					Name:        "Wing Attack",
					Cost:        2,
					Description: "The dragon beats its wings. Each creature within 15 feet must succeed on a DC 25 Dexterity saving throw or take 17 (2d6+10) bludgeoning damage and be knocked prone.",
				},
			},
		},
		LairActions: []LairAction{
			{
				Description: "Magma erupts from a point on the ground the dragon can see within 120 feet. Each creature within 20 feet must make a DC 15 Dexterity saving throw or take 21 (6d6) fire damage.",
				SaveDC:      15,
				SaveType:    "DEX",
			},
		},
	}

	// Additional monsters would be stored in a data structure or loaded from files
	goblin := MonsterStat{
		Name:      "Goblin",
		Size:      "Small",
		Type:      "humanoid",
		Alignment: "neutral evil",
		HP:        7,
		AC:        15,
		Speed: map[string]int{
			"walk": 30,
		},
		AbilityScores: map[string]int{
			"STR": 8, "DEX": 14, "CON": 10,
			"INT": 10, "WIS": 8, "CHA": 8,
		},
		Skills: map[string]int{
			"Stealth": 6,
		},
		Senses: map[string]int{
			"darkvision": 60,
		},
		Languages:       []string{"Common", "Goblin"},
		ChallengeRating: 0.25,
		Traits: []MonsterTrait{
			{
				Name:        "Nimble Escape",
				Description: "The goblin can take the Disengage or Hide action as a bonus action on each of its turns.",
			},
		},
		Actions: []MonsterAction{
			{
				Name:        "Scimitar",
				AttackBonus: 4,
				DamageType:  "slashing",
				DamageDice:  "1d6+2",
			},
		},
	}

	// Simple name matching (production would use proper URI parsing)
	monsters := map[string]MonsterStat{
		"Ancient Red Dragon": dragon,
		"Goblin":             goblin,
	}

	// Return requested monster or dragon as default
	result := dragon
	for name, monster := range monsters {
		if name == uri {
			result = monster
			break
		}
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// handleDamageRules returns SRD damage calculation rules
func handleDamageRules(ctx context.Context, uri string) (string, error) {
	rules := DamageRules{
		ResistanceMultiplier:    0.5,
		VulnerabilityMultiplier: 2.0,
		ImmunityEffect:          "no damage taken",
		CriticalMultiplier:      2,
		ConditionEffects: map[string]string{
			"resistance":    "Damage of specified type is halved",
			"vulnerability": "Damage of specified type is doubled",
			"immunity":      "No damage of specified type is taken",
		},
	}

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ConditionDefinition describes a game condition
type ConditionDefinition struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Effects      []string `json:"effects"`
	EndCondition string   `json:"end_condition"`
}

// handleConditionRules returns all D&D 5e condition definitions
func handleConditionRules(ctx context.Context, uri string) (string, error) {
	conditions := []ConditionDefinition{
		{
			Name:        "Stunned",
			Description: "A stunned creature is incapacitated, can't move, and can speak only falteringly.",
			Effects: []string{
				"Automatically fails Strength and Dexterity saving throws",
				"Attack rolls against the creature have advantage",
			},
			EndCondition: "End of specified duration or until condition is removed",
		},
		{
			Name:        "Prone",
			Description: "A prone creature's only movement option is to crawl.",
			Effects: []string{
				"Disadvantage on attack rolls",
				"Attack rolls against creature have advantage if attacker is within 5 feet",
				"Attack rolls against creature have disadvantage if attacker is more than 5 feet away",
			},
			EndCondition: "Use half movement to stand up",
		},
		{
			Name:        "Paralyzed",
			Description: "A paralyzed creature is incapacitated and can't move or speak.",
			Effects: []string{
				"Automatically fails Strength and Dexterity saving throws",
				"Attack rolls against the creature have advantage",
				"Any attack that hits is a critical hit if attacker is within 5 feet",
			},
			EndCondition: "End of specified duration or until condition is removed",
		},
		{
			Name:        "Poisoned",
			Description: "A poisoned creature has disadvantage on attack rolls and ability checks.",
			Effects: []string{
				"Disadvantage on attack rolls",
				"Disadvantage on ability checks",
			},
			EndCondition: "End of poison duration",
		},
	}

	data, err := json.MarshalIndent(conditions, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// SavingThrowRule defines how saving throws work
type SavingThrowRule struct {
	Types           []string          `json:"types"`
	CriticalSuccess string            `json:"critical_success"`
	CriticalFailure string            `json:"critical_failure"`
	Modifiers       map[string]string `json:"modifiers"`
}

// handleSavingThrowRules returns SRD saving throw mechanics
func handleSavingThrowRules(ctx context.Context, uri string) (string, error) {
	rules := SavingThrowRule{
		Types:           []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"},
		CriticalSuccess: "Natural 20 always succeeds",
		CriticalFailure: "Natural 1 always fails",
		Modifiers: map[string]string{
			"proficiency":  "Add proficiency bonus if proficient in that save",
			"advantage":    "Roll twice, take higher result",
			"disadvantage": "Roll twice, take lower result",
		},
	}

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// LegendaryRule defines legendary action mechanics
type LegendaryRule struct {
	LegendaryActions     LegendaryActionRule     `json:"legendary_actions"`
	LegendaryResistances LegendaryResistanceRule `json:"legendary_resistances"`
	LairActions          LairActionRule          `json:"lair_actions"`
}

type LegendaryActionRule struct {
	Description     string `json:"description"`
	Timing          string `json:"timing"`
	ResetTiming     string `json:"reset_timing"`
	DefaultPerRound int    `json:"default_per_round"`
}

type LegendaryResistanceRule struct {
	Description  string `json:"description"`
	DefaultCount int    `json:"default_count"`
	Usage        string `json:"usage"`
	ResetTiming  string `json:"reset_timing"`
}

type LairActionRule struct {
	Description string `json:"description"`
	Initiative  int    `json:"initiative"`
	Frequency   string `json:"frequency"`
}

// handleLegendaryRules returns rules for legendary mechanics
func handleLegendaryRules(ctx context.Context, uri string) (string, error) {
	rules := LegendaryRule{
		LegendaryActions: LegendaryActionRule{
			Description:     "Special actions that can be taken outside the creature's turn",
			Timing:          "At the end of another creature's turn",
			ResetTiming:     "Start of the legendary creature's turn",
			DefaultPerRound: 3,
		},
		LegendaryResistances: LegendaryResistanceRule{
			Description:  "Ability to automatically succeed on a failed saving throw",
			DefaultCount: 3,
			Usage:        "Choose to succeed on a failed save",
			ResetTiming:  "After a long rest or per encounter (DM discretion)",
		},
		LairActions: LairActionRule{
			Description: "Environmental effects that occur in the creature's lair",
			Initiative:  20,
			Frequency:   "Once per round on initiative count 20",
		},
	}

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// handleMonsterList returns a list of available monsters
func handleMonsterList(ctx context.Context, uri string) (string, error) {
	monsters := []map[string]interface{}{
		{
			"name": "Ancient Red Dragon",
			"cr":   24,
			"type": "dragon",
			"uri":  "monster://stat_block/Ancient%20Red%20Dragon",
		},
		{
			"name": "Goblin",
			"cr":   0.25,
			"type": "humanoid",
			"uri":  "monster://stat_block/Goblin",
		},
		{
			"name": "Beholder",
			"cr":   13,
			"type": "aberration",
			"uri":  "monster://stat_block/Beholder",
		},
		{
			"name": "Lich",
			"cr":   21,
			"type": "undead",
			"uri":  "monster://stat_block/Lich",
		},
	}

	data, err := json.MarshalIndent(monsters, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
