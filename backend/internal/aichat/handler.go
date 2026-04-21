package aichat

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

// ── Request / Response types ─────────────────────────────────────────────────

type ChatMessage struct {
	Role    string `json:"role"`    // "user" | "assistant" | "system"
	Content string `json:"content"`
}

type ChatRequest struct {
	Message string        `json:"message" binding:"required"`
	History []ChatMessage `json:"history"`
}

type ProductSuggestion struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Reason string  `json:"reason"`
}

type SuggestedAction struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	URL   string `json:"url,omitempty"`
}

type ChatResponse struct {
	Message     string              `json:"message"`
	Suggestions []ProductSuggestion `json:"suggestions,omitempty"`
	Actions     []SuggestedAction   `json:"actions,omitempty"`
}

// ── OpenAI types ─────────────────────────────────────────────────────────────

type openAIRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// POST /api/v1/ai/chat
func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "message is required")
		return
	}

	if len(req.Message) > 1000 {
		response.BadRequest(c, "message too long (max 1000 chars)")
		return
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	var reply ChatResponse

	if apiKey != "" {
		reply = callOpenAI(apiKey, req)
	} else {
		reply = ruleBasedResponse(req.Message)
	}

	response.OK(c, reply)
}

// ── OpenAI integration ───────────────────────────────────────────────────────

const systemPrompt = `You are a helpful AI shopping assistant for GeoCore, a marketplace for buying and selling items and auctions.
Help users find products, answer questions about how the platform works, guide them through buying/selling, and provide friendly support.
Keep responses concise (2-4 sentences max). If asked about specific products, suggest browsing the listings page.
Respond in the same language the user writes in (Arabic or English).`

func callOpenAI(apiKey string, req ChatRequest) ChatResponse {
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	for _, m := range req.History {
		if m.Role == "user" || m.Role == "assistant" {
			messages = append(messages, m)
		}
	}
	messages = append(messages, ChatMessage{Role: "user", Content: req.Message})

	body, _ := json.Marshal(openAIRequest{
		Model:     "gpt-3.5-turbo",
		Messages:  messages,
		MaxTokens: 300,
	})

	httpClient := &http.Client{Timeout: 15 * time.Second}
	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ruleBasedResponse(req.Message)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ruleBasedResponse(req.Message)
	}
	defer resp.Body.Close()

	var oaiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaiResp); err != nil || len(oaiResp.Choices) == 0 {
		return ruleBasedResponse(req.Message)
	}

	return ChatResponse{
		Message: oaiResp.Choices[0].Message.Content,
		Actions: defaultActions(req.Message),
	}
}

// ── Rule-based fallback ───────────────────────────────────────────────────────

func ruleBasedResponse(msg string) ChatResponse {
	lower := strings.ToLower(msg)

	switch {
	case contains(lower, "buy", "شراء", "أشتري", "purchase"):
		return ChatResponse{
			Message: "Welcome! To buy on GeoCore, browse our listings or auctions. You can filter by category, price, or location. Once you find what you like, add it to cart and checkout securely.",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "Browse Listings", URL: "/listings"},
				{Type: "navigate", Label: "Live Auctions", URL: "/auctions"},
			},
		}
	case contains(lower, "sell", "بيع", "أبيع", "listing"):
		return ChatResponse{
			Message: "To sell on GeoCore, create a listing with photos, description, and price. You can also create auction listings for higher-value items. Your items will be visible to thousands of buyers!",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "Create Listing", URL: "/sell"},
			},
		}
	case contains(lower, "auction", "مزاد", "مزايدة"):
		return ChatResponse{
			Message: "Our live auctions let you bid on unique items in real-time. Set your maximum bid and we'll auto-bid for you up to that amount. Auctions end at the scheduled time and the highest bidder wins!",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "View Auctions", URL: "/auctions"},
			},
		}
	case contains(lower, "payment", "pay", "دفع", "stripe", "paypal"):
		return ChatResponse{
			Message: "GeoCore supports secure payments via Stripe (credit/debit cards) and PayPal. All payments are held in escrow until you confirm delivery — your money is always protected.",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "Buyer Protection", URL: "/buyer-protection"},
			},
		}
	case contains(lower, "order", "طلب", "orders"):
		return ChatResponse{
			Message: "You can track all your orders from the Orders page. Once a seller ships your item, you'll get a tracking number. Confirm delivery when you receive it to release payment to the seller.",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "My Orders", URL: "/orders"},
			},
		}
	case contains(lower, "refund", "dispute", "problem", "issue", "مشكلة", "استرداد"):
		return ChatResponse{
			Message: "If you have an issue with an order, you can open a dispute from your order page. Our team reviews all disputes and ensures fair resolution. Most disputes are resolved within 3-5 business days.",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "Open Dispute", URL: "/disputes/new"},
				{Type: "navigate", Label: "Contact Support", URL: "/contact"},
			},
		}
	case contains(lower, "account", "profile", "حساب", "ملف"):
		return ChatResponse{
			Message: "Manage your account from the Profile page — update your photo, contact info, and notification preferences. You can also view your loyalty points and referral stats there.",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "My Profile", URL: "/profile"},
			},
		}
	case contains(lower, "help", "مساعدة", "كيف", "how"):
		return ChatResponse{
			Message: "I'm here to help! You can ask me about buying, selling, payments, orders, or how the platform works. For detailed guides, check our Help Center.",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "Help Center", URL: "/help"},
				{Type: "navigate", Label: "FAQ", URL: "/help/faq"},
			},
		}
	case contains(lower, "fee", "commission", "رسوم", "عمولة"):
		return ChatResponse{
			Message: "GeoCore charges a small platform fee only on successful sales — no listing fees! Use our Fee Calculator to see exactly what you'll earn from a sale.",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "Fee Calculator", URL: "/fees/calculator"},
				{Type: "navigate", Label: "Fee Schedule", URL: "/fees"},
			},
		}
	default:
		return ChatResponse{
			Message: "Hi there! 👋 I'm your GeoCore shopping assistant. I can help you with buying, selling, auctions, payments, orders, and more. What would you like to know?",
			Actions: []SuggestedAction{
				{Type: "navigate", Label: "Browse Listings", URL: "/listings"},
				{Type: "navigate", Label: "Help Center", URL: "/help"},
			},
		}
	}
}

func defaultActions(msg string) []SuggestedAction {
	lower := strings.ToLower(msg)
	if contains(lower, "buy", "listing", "شراء") {
		return []SuggestedAction{{Type: "navigate", Label: "Browse Listings", URL: "/listings"}}
	}
	if contains(lower, "sell", "بيع") {
		return []SuggestedAction{{Type: "navigate", Label: "Create Listing", URL: "/sell"}}
	}
	return nil
}

func contains(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
