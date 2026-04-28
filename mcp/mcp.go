// Package mcp exposes the discord-go [discord.Client] surface as a set
// of MCP (Model Context Protocol) tools.
//
// Tools are defined via [mcptool.Define] so JSON input schemas are
// reflected from typed input structs — no hand-maintained schemas, no
// drift. The coverage test in mcp_test.go fails if a new exported
// method is added to *discord.Client without either being wrapped by
// a tool or appearing in [Excluded] (with a reason).
package mcp

import "github.com/teslashibe/mcptool"

// Provider implements [mcptool.Provider] for discord-go.
type Provider struct{}

// Platform returns "discord".
func (Provider) Platform() string { return "discord" }

// Tools returns every discord-go MCP tool, in registration order.
func (Provider) Tools() []mcptool.Tool {
	out := make([]mcptool.Tool, 0,
		len(statusTools)+len(guildTools)+len(channelTools)+
			len(messageTools)+len(searchTools)+len(memberTools)+
			len(sendTools)+len(reactTools)+len(watchTools))
	out = append(out, statusTools...)
	out = append(out, guildTools...)
	out = append(out, channelTools...)
	out = append(out, messageTools...)
	out = append(out, searchTools...)
	out = append(out, memberTools...)
	out = append(out, sendTools...)
	out = append(out, reactTools...)
	out = append(out, watchTools...)
	return out
}
