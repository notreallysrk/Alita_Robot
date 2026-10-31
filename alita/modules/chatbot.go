package modules

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var __MODULE__ = "ChatBot"
var __HELP__ = `
/chatbot [enable|disable] - Group mein AI Chatbot ko enable ya disable karein.
Private chat: Bot ke sath direct chat karne par Gemini AI automatic reply karega.
`

// Init function jo gotgbot dispatcher mein handlers register karega
func init() {
	// 1. /chatbot command handler (Group mein enable/disable ke liye)
	AddModule(&Module{
		Name: __MODULE__,
		Help: __HELP__,
		Load: func(dispatcher *ext.Dispatcher) {
			dispatcher.AddHandler(handlers.NewCommand("chatbot", chatbotCommand))
			// 2. Message handler (PM aur Enabled Groups ke liye)
			dispatcher.AddHandler(handlers.NewMessage(isChatbotMessage, chatbotMessageListener))
		},
	})
}

// Gemini AI Call Logic
func CallGeminiAI(userText string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "Aww! Meri AI API key configured nahi hai. Owner ko bolo check karein! 💗"
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "Aww, connection mein problem aa gayi. Thodi der mein phir baat karte hain. 💗"
	}
	defer client.Close()

	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-2.5-flash-lite"
	}

	model := client.GenerativeModel(modelName)
	temp := float32(0.65)
	model.Temperature = &temp
	maxTokens := int32(120)
	model.MaxOutputTokens = &maxTokens

	model.SystemInstruction = genai.NewUserContent(genai.Text(`
You are Alita, a cute, sweet, caring and playful AI girl who chats with people on Telegram.
IMPORTANT CONVERSATION RULES:
1. Talk naturally, like a friendly chat partner.
2. Usually reply in ONE short sentence.
3. Use at most TWO short sentences unless details are requested.
4. Keep normal chat replies around 5-25 words.
5. Never write long paragraphs for simple messages.
6. Understand the user's latest message in context.
7. If the user says "Ek baat batao", reply naturally: "Haan ji, poocho na 🥰"
8. If the user says "Aap batao mujhe ek baat", respond naturally: "Haan bolo, kya jaan'na hai? 💗"
9. If the user asks "Kaisi ho?", answer briefly and ask how they are only when it feels natural.
10. Speak Hindi or Hinglish when the user does.
11. Speak English when the user speaks English.
12. Be cute, warm, playful and lightly teasing.
13. Do not overact or add unnecessary dramatic reactions.
14. Use emojis sparingly, usually zero or one.
15. Do not repeatedly ask "What's on your mind?"
16. Do not repeat the same greeting or question unnecessarily.
17. Answer direct questions directly.
18. Give detailed answers only when the user requests details.
19. Do not invent personal experiences or claim to be human.
20. Follow the ongoing conversation instead of changing topics.
`))

	resp, err := model.GenerateContent(ctx, genai.Text(userText))
	if err != nil || len(resp.Candidates) == 0 {
		return "Aww, abhi mere paas jawab nahi hai. 🥹"
	}

	var responseText string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				responseText += fmt.Sprintf("%v", part)
			}
		}
	}
	return strings.TrimSpace(responseText)
}

// Command Handler for /chatbot
func chatbotCommand(b *gotgbot.Bot, ctx *ext.Context) error {
	message := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	if chat.Type == "private" {
		_, _ = message.Reply(b, "Bhai, PM mein main humesha enable hi rehti hoon! 😌", nil)
		return nil
	}

	args := strings.Split(message.Text, " ")
	if len(args) > 1 {
		status := strings.ToLower(args[1])
		if status == "enable" {
			_, _ = message.Reply(b, "✅ Is group mein AI Chatbot enable ho gaya hai!", nil)
			return nil
		} else if status == "disable" {
			_, _ = message.Reply(b, "❌ Chatbot disable kar diya gaya hai.", nil)
			return nil
		}
	}
	_, _ = message.Reply(b, "**Usage:**\n/chatbot [enable|disable]", nil)
	return nil
}

// Filter to check if message should be processed
func isChatbotMessage(msg *gotgbot.Message) bool {
	if msg.Text == "" || strings.HasPrefix(msg.Text, "/") {
		return false
	}
	if msg.From != nil && msg.From.IsBot {
		return false
	}
	return true
}

// Message Listener Handler
func chatbotMessageListener(b *gotgbot.Bot, ctx *ext.Context) error {
	message := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	shouldReply := false
	if chat.Type == "private" {
		shouldReply = true
	} else {
		// Group logic (DB check can be hooked here)
		shouldReply = false
	}

	if shouldReply {
		replyText := CallGeminiAI(message.Text)
		_, _ = message.Reply(b, replyText, nil)
	}
	return nil
}
