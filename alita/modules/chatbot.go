package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

var __MODULE__ = "ChatBot"
var __HELP__ = `
/chatbot [enable|disable] - Group mein AI Chatbot ko enable ya disable karein.
Private chat: Bot ke sath direct chat karne par Gemini AI automatic reply karega.
`

func init() {
	AddModule(&Module{
		Name: __MODULE__,
		Help: __HELP__,
		Load: func(dispatcher *ext.Dispatcher) {
			dispatcher.AddHandler(handlers.NewCommand("chatbot", chatbotCommand))
			dispatcher.AddHandler(handlers.NewMessage(isChatbotMessage, chatbotMessageListener))
		},
	})
}

// Direct HTTP Request to Gemini API (No external genai package required)
func CallGeminiAI(userText string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("GEMINI_API_KEY is not configured.")
		return "Aww! Meri AI API key configured nahi hai. Owner ko bolo check karein! 💗"
	}

	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-2.5-flash-lite"
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)

	personality := `You are Alita, a cute, sweet, caring and playful AI girl who chats with people on Telegram.
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
20. Follow the ongoing conversation instead of changing topics.`

	requestBody := map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": personality},
			},
		},
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": userText},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.65,
			"maxOutputTokens": 120,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("JSON Marshal Error: %v", err)
		return "Aww, abhi mere paas jawab nahi hai. 🥹"
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Gemini HTTP Error: %v", err)
		return "Aww, connection mein problem aa gayi. Thodi der mein phir baat karte hain. 💗"
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("JSON Decode Error: %v", err)
		return "Aww, samajh nahi aaya kya bolun. 🦊"
	}

	// Extract text from response candidates
	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "Aww, abhi mere paas jawab nahi hai. 🥹"
	}

	candidate, ok := candidates[0].(map[string]interface{})
	if !ok {
		return "Aww, abhi mere paas jawab nahi hai. 🥹"
	}

	content, ok := candidate["content"].(map[string]interface{})
	if !ok {
		return "Aww, abhi mere paas jawab nahi hai. 🥹"
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "Aww, abhi mere paas jawab nahi hai. 🥹"
	}

	part, ok := parts[0].(map[string]interface{})
	if !ok {
		return "Aww, abhi mere paas jawab nahi hai. 🥹"
	}

	text, ok := part["text"].(string)
	if !ok {
		return "Aww, abhi mere paas jawab nahi hai. 🥹"
	}

	return strings.TrimSpace(text)
}

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

func isChatbotMessage(msg *gotgbot.Message) bool {
	if msg.Text == "" || strings.HasPrefix(msg.Text, "/") {
		return false
	}
	if msg.From != nil && msg.From.IsBot {
		return false
	}
	return true
}

func chatbotMessageListener(b *gotgbot.Bot, ctx *ext.Context) error {
	message := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	shouldReply := false
	if chat.Type == "private" {
		shouldReply = true
	} else {
		shouldReply = false
	}

	if shouldReply {
		replyText := CallGeminiAI(message.Text)
		_, _ = message.Reply(b, replyText, nil)
	}
	return nil
}
