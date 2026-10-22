package models

// Estructuras para el payload de Discord
type DiscordPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds"`
}

type DiscordEmbed struct {
	Title  string       `json:"title,omitempty"`
	URL    string       `json:"url,omitempty"`   // Hace el título clickeable
	Color  int          `json:"color,omitempty"` // Decimal color
	Fields []EmbedField `json:"fields,omitempty"`
	Footer *EmbedFooter `json:"footer,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type EmbedFooter struct {
	Text string `json:"text"`
}
