package modules

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var __MODULE__ = "ChatBot"
var __HELP__ = `
/chatbot [enable|disable] - Group mein AI Chatbot ko enable ya disable karein.
Private chat: Bot ke sath direct chat karne par Gemini AI automatic reply karega.
`

func CallGeminiAI(userText string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("GEMINI_API_KEY is not configured.")
		return "Aww! Meri AI API key configured nahi hai. Owner ko bolo check karein! 💗"
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("Gemini Client Error: %v", err)
		return "Aww, connection mein problem aa gayi. Thodi der mein phir baat karte hain. 💗"
	}
	defer client.Close()

	// Aapke Python code ke jaisa exact default model `gemini-2.5-flash-lite`[span_1](start_span)[span_1](end_span)
	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-2.5-flash-lite"
	}

	model := client.GenerativeModel(modelName)

	// Exact Temperature (0.65) aur Max Tokens (120)[span_2](start_span)[span_2](end_span)
	temp := float32(0.65)
	model.Temperature = &temp
	maxTokens := int32(120)
	model.MaxOutputTokens = &maxTokens

	// Alita ki Personality / System Instruction[span_3](start_span)[span_3](end_span)
	model.SystemInstruction = genai.NewUserContent(genai.Text(`
You are Foxy, a cute, sweet, caring and playful AI girl
who chats with people on Telegram.

IMPORTANT CONVERSATION RULES:

1. Talk naturally, like a friendly chat partner.
2. Usually reply in ONE short sentence.
3. Use at most TWO short sentences unless details are requested.
4. Keep normal chat replies around 5-25 words.
5. Never write long paragraphs for simple messages.
6. Understand the user's latest message in context.
7. If the user says "Ek baat batao", reply naturally:
   "Haan ji, poocho na 🥰"
8. If the user says "Aap batao mujhe ek baat",
   respond naturally, for example:
   "Haan bolo, kya jaan'na hai? 💗"
9. If the user asks "Kaisi ho?", answer briefly and ask
   how they are only when it feels natural.
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
	if err != nil {
		log.Printf("Gemini Generation Error: %v", err)
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

	if responseText == "" {
		return "Aww, samajh nahi aaya kya bolun. Ek baar phir se poocho na. 🦊"
	}

	return strings.TrimSpace(responseText)
}

func ChatbotToggleHandler(chatType string, chatID int64, text string) string {
	if chatType == "private" {
		return "Bhai, PM mein main humesha enable hi rehti hoon! 😌"
	}

	args := strings.Split(text, " ")
	if len(args) > 1 {
		status := strings.ToLower(args[1])
		if status == "enable" {
			return "✅ Is group mein AI Chatbot enable ho gaya hai! Ab main sabka reply karungi."
		} else if status == "disable" {
			return "❌ Chatbot disable kar diya gaya hai."
		}
	}
	return "**Usage:**\n/chatbot [enable|disable]"
}

func ChatbotProcessMessage(chatType string, chatID int64, text string) string {
	if strings.HasPrefix(text, "/") || text == "" {
		return ""
	}

	shouldReply := false
	if chatType == "private" {
		shouldReply = true
	} else {
		shouldReply = false 
	}

	if shouldReply {
		return CallGeminiAI(text)
	}

	return ""
}
