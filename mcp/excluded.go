package mcp

// Excluded enumerates exported methods on *discord.Client that are
// intentionally not exposed via MCP. Each entry must have a non-empty
// reason. The coverage test fails if a new exported method is added
// without either being wrapped by a tool or appearing here, or if an
// entry here doesn't correspond to a real method.
var Excluded = map[string]string{
	"Close":      "lifecycle method owned by the host application; not a callable agent tool",
	"Connect":    "host wires this once at startup with the bot token + intents; not exposed as a runtime tool",
	"Disconnect": "lifecycle method owned by the host application; not a callable agent tool",
}
