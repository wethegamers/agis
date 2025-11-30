// Package scripting provides a Tengo-based scripting engine for custom commands.
// This allows users to define simple bot commands in Tengo scripts that can be
// hot-reloaded without rebuilding the bot.
package scripting

import (
	"context"
	"fmt"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// ScriptContext provides the execution context for scripts
type ScriptContext struct {
	// User info
	UserID        string
	Username      string
	Discriminator string
	IsPremium     bool
	IsVerified    bool
	IsMod         bool
	IsAdmin       bool

	// Guild info
	GuildID     string
	GuildName   string
	ChannelID   string
	ChannelName string

	// Command info
	CommandName string
	Args        []string
	RawContent  string

	// User economy data
	Credits     int
	WTGCoins    int
	Tier        string
	ServerCount int
}

// ScriptResult is the result of script execution
type ScriptResult struct {
	// Response text (plain text or markdown)
	Text string

	// Embed data (if returning an embed)
	Embed *EmbedData

	// Whether to reply privately (DM)
	Private bool

	// Error message (if script failed)
	Error string

	// Side effects
	CreditChange int    // Positive = add, negative = subtract
	LogMessage   string // Message to log
}

// EmbedData represents a Discord embed
type EmbedData struct {
	Title       string
	Description string
	Color       int
	URL         string
	Fields      []EmbedField
	Footer      string
	Thumbnail   string
	Image       string
}

// EmbedField represents an embed field
type EmbedField struct {
	Name   string
	Value  string
	Inline bool
}

// Engine executes Tengo scripts for custom commands
type Engine struct {
	// Timeout for script execution
	Timeout time.Duration

	// Maximum allowed instructions
	MaxInstructions int64
}

// NewEngine creates a new scripting engine
func NewEngine() *Engine {
	return &Engine{
		Timeout:         5 * time.Second,
		MaxInstructions: 10000,
	}
}

// Execute runs a Tengo script with the given context
func (e *Engine) Execute(ctx context.Context, script string, sctx *ScriptContext) (*ScriptResult, error) {
	// Create script with stdlib modules
	s := tengo.NewScript([]byte(script))

	// Add safe stdlib modules (no os, no exec)
	s.SetImports(stdlib.GetModuleMap(
		"fmt",
		"text",
		"times",
		"math",
		"rand",
		"json",
	))

	// Set resource limits
	s.SetMaxAllocs(e.MaxInstructions)

	// Add context variables
	if err := e.setContextVariables(s, sctx); err != nil {
		return nil, fmt.Errorf("failed to set context: %w", err)
	}

	// Add result variables
	_ = s.Add("result_text", "")
	_ = s.Add("result_error", "")
	_ = s.Add("result_private", false)
	_ = s.Add("result_credit_change", 0)
	_ = s.Add("result_log", "")

	// Add embed builder
	_ = s.Add("result_embed_title", "")
	_ = s.Add("result_embed_description", "")
	_ = s.Add("result_embed_color", 0)
	_ = s.Add("result_embed_footer", "")
	_ = s.Add("result_embed_fields", []interface{}{})

	// Compile with timeout
	compiled, err := s.Compile()
	if err != nil {
		return nil, fmt.Errorf("script compilation error: %w", err)
	}

	// Run with timeout
	runCtx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()

	if err := compiled.RunContext(runCtx); err != nil {
		return nil, fmt.Errorf("script execution error: %w", err)
	}

	// Extract results
	result := &ScriptResult{}

	if v := compiled.Get("result_text"); v != nil {
		result.Text = v.String()
	}
	if v := compiled.Get("result_error"); v != nil && v.String() != "" {
		result.Error = v.String()
	}
	if v := compiled.Get("result_private"); v != nil {
		if boolVal, ok := v.Value().(bool); ok {
			result.Private = boolVal
		}
	}
	if v := compiled.Get("result_credit_change"); v != nil {
		if intVal, ok := v.Value().(int64); ok {
			result.CreditChange = int(intVal)
		} else if intVal, ok := v.Value().(int); ok {
			result.CreditChange = intVal
		}
	}
	if v := compiled.Get("result_log"); v != nil {
		result.LogMessage = v.String()
	}

	// Extract embed if set
	embedTitle := compiled.Get("result_embed_title")
	if embedTitle != nil && embedTitle.String() != "" {
		result.Embed = &EmbedData{
			Title: embedTitle.String(),
		}
		if v := compiled.Get("result_embed_description"); v != nil {
			result.Embed.Description = v.String()
		}
		if v := compiled.Get("result_embed_color"); v != nil {
			if intVal, ok := v.Value().(int64); ok {
				result.Embed.Color = int(intVal)
			} else if intVal, ok := v.Value().(int); ok {
				result.Embed.Color = intVal
			}
		}
		if v := compiled.Get("result_embed_footer"); v != nil {
			result.Embed.Footer = v.String()
		}
		// Parse embed fields
		if v := compiled.Get("result_embed_fields"); v != nil {
			if arr, ok := v.Value().([]interface{}); ok {
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						field := EmbedField{}
						if name, ok := m["name"].(string); ok {
							field.Name = name
						}
						if value, ok := m["value"].(string); ok {
							field.Value = value
						}
						if inline, ok := m["inline"].(bool); ok {
							field.Inline = inline
						}
						result.Embed.Fields = append(result.Embed.Fields, field)
					}
				}
			}
		}
	}

	return result, nil
}

func (e *Engine) setContextVariables(s *tengo.Script, sctx *ScriptContext) error {
	// User info
	_ = s.Add("user_id", sctx.UserID)
	_ = s.Add("user_name", sctx.Username)
	_ = s.Add("user_discriminator", sctx.Discriminator)
	_ = s.Add("user_is_premium", sctx.IsPremium)
	_ = s.Add("user_is_verified", sctx.IsVerified)
	_ = s.Add("user_is_mod", sctx.IsMod)
	_ = s.Add("user_is_admin", sctx.IsAdmin)

	// Guild info
	_ = s.Add("guild_id", sctx.GuildID)
	_ = s.Add("guild_name", sctx.GuildName)
	_ = s.Add("channel_id", sctx.ChannelID)
	_ = s.Add("channel_name", sctx.ChannelName)

	// Command info
	_ = s.Add("command_name", sctx.CommandName)
	_ = s.Add("args", sctx.Args)
	_ = s.Add("raw_content", sctx.RawContent)
	_ = s.Add("arg_count", len(sctx.Args))

	// Economy data
	_ = s.Add("user_credits", sctx.Credits)
	_ = s.Add("user_wtg_coins", sctx.WTGCoins)
	_ = s.Add("user_tier", sctx.Tier)
	_ = s.Add("user_server_count", sctx.ServerCount)

	return nil
}

// ValidateScript checks if a script is valid without executing it
func (e *Engine) ValidateScript(script string) error {
	s := tengo.NewScript([]byte(script))
	s.SetImports(stdlib.GetModuleMap(
		"fmt",
		"text",
		"times",
		"math",
		"rand",
		"json",
	))

	// Add dummy context variables for validation
	_ = s.Add("user_id", "")
	_ = s.Add("user_name", "")
	_ = s.Add("user_discriminator", "")
	_ = s.Add("user_is_premium", false)
	_ = s.Add("user_is_verified", false)
	_ = s.Add("user_is_mod", false)
	_ = s.Add("user_is_admin", false)
	_ = s.Add("guild_id", "")
	_ = s.Add("guild_name", "")
	_ = s.Add("channel_id", "")
	_ = s.Add("channel_name", "")
	_ = s.Add("command_name", "")
	_ = s.Add("args", []interface{}{})
	_ = s.Add("raw_content", "")
	_ = s.Add("arg_count", 0)
	_ = s.Add("user_credits", 0)
	_ = s.Add("user_wtg_coins", 0)
	_ = s.Add("user_tier", "")
	_ = s.Add("user_server_count", 0)
	_ = s.Add("result_text", "")
	_ = s.Add("result_error", "")
	_ = s.Add("result_private", false)
	_ = s.Add("result_credit_change", 0)
	_ = s.Add("result_log", "")
	_ = s.Add("result_embed_title", "")
	_ = s.Add("result_embed_description", "")
	_ = s.Add("result_embed_color", 0)
	_ = s.Add("result_embed_footer", "")
	_ = s.Add("result_embed_fields", []interface{}{})

	_, err := s.Compile()
	return err
}
