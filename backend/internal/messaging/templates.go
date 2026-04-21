package messaging

// ── Message Templates ─────────────────────────────────────────────────────────────
//
// Predefined templates for each message type.
// Variables: {{item_name}}, {{price}}, {{user_name}}, {{time_left}}, {{discount}}

// Template defines a message template.
type Template struct {
	Type    string `json:"type"`    // nudge, reminder, win, loss, promo
	Channel string `json:"channel"` // push, email, in_app
	Title   string `json:"title"`
	Body    string `json:"body"`
}

// DefaultTemplates contains all built-in message templates.
var DefaultTemplates = []Template{
	// ── Nudge (come back) ──────────────────────────────────────────────────────
	{Type: "nudge", Channel: ChannelPush, Title: "We miss you! 🎯",
		Body: "New items just listed in your favorite categories. Come check them out!"},
	{Type: "nudge", Channel: ChannelInApp, Title: "Still browsing?",
		Body: "Take a break? We'll save your spot."},

	// ── Reminder ────────────────────────────────────────────────────────────────
	{Type: "reminder", Channel: ChannelPush, Title: "⏰ Auction ending soon!",
		Body: "{{item_name}} ends in {{time_left}}. Don't miss out!"},
	{Type: "reminder", Channel: ChannelEmail, Title: "Your saved item is about to end",
		Body: "Hi {{user_name}}, {{item_name}} auction ends in {{time_left}}. Current price: {{price}}."},
	{Type: "reminder", Channel: ChannelInApp, Title: "Ending soon",
		Body: "{{item_name}} — {{time_left}} left!"},

	// ── Win (dopamine boost) ────────────────────────────────────────────────────
	{Type: "win", Channel: ChannelPush, Title: "🎉 You won!",
		Body: "Congratulations! You won {{item_name}} for {{price}}!"},
	{Type: "win", Channel: ChannelEmail, Title: "You won the auction!",
		Body: "Hi {{user_name}}, you won {{item_name}} for {{price}}! Complete your purchase now."},
	{Type: "win", Channel: ChannelInApp, Title: "🏆 Winner!",
		Body: "You won {{item_name}}!"},

	// ── Loss (re-engagement) ────────────────────────────────────────────────────
	{Type: "loss", Channel: ChannelPush, Title: "You were outbid 😔",
		Body: "Someone bid higher on {{item_name}}. Bid again?"},
	{Type: "loss", Channel: ChannelInApp, Title: "Outbid",
		Body: "{{item_name}} — bid again before it ends!"},
	{Type: "loss", Channel: ChannelPush, Title: "Try again 💪",
		Body: "You lost {{item_name}}, but there are more great deals waiting!"},

	// ── Promo ────────────────────────────────────────────────────────────────────
	{Type: "promo", Channel: ChannelPush, Title: "🔥 Special offer!",
		Body: "{{discount}} off your next purchase. Limited time!"},
	{Type: "promo", Channel: ChannelEmail, Title: "Exclusive deal inside",
		Body: "Hi {{user_name}}, enjoy {{discount}} off your next purchase. Offer expires soon!"},
	{Type: "promo", Channel: ChannelInApp, Title: "Special deal",
		Body: "{{discount}} off — limited time!"},
}

// GetTemplate finds a template by type and channel.
func GetTemplate(msgType, channel string) *Template {
	for _, t := range DefaultTemplates {
		if t.Type == msgType && t.Channel == channel {
			return &t
		}
	}
	// Fallback: return first matching type
	for _, t := range DefaultTemplates {
		if t.Type == msgType {
			return &t
		}
	}
	return nil
}
