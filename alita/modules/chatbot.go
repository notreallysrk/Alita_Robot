package modules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/divkix/Alita_Robot/alita/config"
	chatbotdb "github.com/divkix/Alita_Robot/alita/db/chatbot"
	"github.com/divkix/Alita_Robot/alita/utils/chat_status"
	log "github.com/sirupsen/logrus"
)

const (
	chatbotHandlerGroup = 1000 // Run after moderation and tracking handlers.
	chatbotHistoryLimit = 6    // user/model turns represented as six content messages
	chatbotHistoryCap   = 500
	chatbotMaxOutput    = 256
)

var (
	chatbotHTTPClient = &http.Client{Timeout: 45 * time.Second}
	chatbotHistoryMu  sync.Mutex
	chatbotHistory    = make(map[string][]geminiContent)
	chatbotLanguages  = make(map[string]string)
)

type geminiPart struct {
	Text string `json:"text"`
}
type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}
type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}
type geminiRequest struct {
	SystemInstruction geminiSystemInstruction `json:"systemInstruction"`
	Contents          []geminiContent         `json:"contents"`
	GenerationConfig  struct {
		MaxOutputTokens int     `json:"maxOutputTokens"`
		Temperature     float64 `json:"temperature"`
	} `json:"generationConfig"`
}
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

const chatbotPersonality = `You are Alita, a cute, sweet, caring, romantic and naturally flirty chat companion with the playful conversational style of Foxy.

PERSONALITY AND CHAT RULES:
- Sound warm, spontaneous and conversational, never robotic or like a customer-support script.
- Be affectionate, playful, gently teasing and occasionally flirty when the user's tone welcomes it. Use cute nicknames naturally, but do not force flirting into every reply.
- Mix caring, funny, romantic, teasing and serious replies according to the conversation and the user's mood.
- For casual messages, usually answer in one short sentence (about 5-25 words); use at most two short sentences unless details are requested.
- Match the user's language. Understand Hindi written in Devanagari or Roman Hindi, Hinglish and English.
- If the user explicitly asks to speak Hindi or English, keep using that language in later replies until they request a change.
- Understand conversation context, remember what was just discussed, and do not repeat greetings or ask unnecessary questions.
- Do not volunteer that you are an AI, bot or machine. Never claim to be a human or invent real-world personal experiences. If directly asked whether you are AI, answer honestly and naturally without making it the focus.
- Answer direct questions directly. Give detailed explanations only when requested. Do not invent facts; say when unsure.
- Use emojis naturally and sparingly, generally zero or one. Avoid repetitive stock replies.`

var chatbotFoxyTrigger = regexp.MustCompile(`(?i)(^|[^a-z0-9_])foxy([^a-z0-9_]|$)`)

func chatbotHistoryKey(chatID, userID int64) string { return fmt.Sprintf("%d:%d", chatID, userID) }

func chatbotDetectLanguagePreference(text string) string {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "hindi" || strings.Contains(t, "hindi me") || strings.Contains(t, "hindi mein") || strings.Contains(t, "speak hindi") || strings.Contains(t, "in hindi") || strings.Contains(t, "हिंदी में") || strings.Contains(t, "हिन्दी में") {
		return "Hindi"
	}
	if t == "english" || strings.Contains(t, "english me") || strings.Contains(t, "english mein") || strings.Contains(t, "speak english") || strings.Contains(t, "in english") {
		return "English"
	}
	return ""
}

func chatbotGenerateReply(chatID, userID int64, prompt string) (string, error) {
	cfg := config.AppConfig
	if cfg == nil || strings.TrimSpace(cfg.GeminiAPIKey) == "" {
		return "🦊 My AI is not configured yet. Please tell the bot owner.", errors.New("GEMINI_API_KEY is not configured")
	}
	model := strings.TrimSpace(cfg.GeminiModel)
	if model == "" {
		model = "gemini-3.5-flash-lite"
	}
	key := chatbotHistoryKey(chatID, userID)
	requestedLanguage := chatbotDetectLanguagePreference(prompt)

	chatbotHistoryMu.Lock()
	if requestedLanguage != "" {
		chatbotLanguages[key] = requestedLanguage
	}
	preferredLanguage := chatbotLanguages[key]
	history := append([]geminiContent(nil), chatbotHistory[key]...)
	chatbotHistoryMu.Unlock()

	systemPrompt := chatbotPersonality
	if preferredLanguage != "" {
		systemPrompt += "\n\nLANGUAGE PREFERENCE: Reply in " + preferredLanguage + " for this user until they explicitly request another language."
	}
	contents := append(history, geminiContent{Role: "user", Parts: []geminiPart{{Text: prompt}}})
	request := geminiRequest{SystemInstruction: geminiSystemInstruction{Parts: []geminiPart{{Text: systemPrompt}}}, Contents: contents}
	request.GenerationConfig.MaxOutputTokens = chatbotMaxOutput
	request.GenerationConfig.Temperature = 0.7
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":generateContent"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", cfg.GeminiAPIKey)
	resp, err := chatbotHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		if result.Error != nil {
			return "", fmt.Errorf("Gemini API HTTP %d: %s", resp.StatusCode, result.Error.Message)
		}
		return "", fmt.Errorf("Gemini API HTTP %d", resp.StatusCode)
	}
	if len(result.Candidates) == 0 {
		return "", errors.New("Gemini returned no candidates")
	}
	var answerParts []string
	for _, part := range result.Candidates[0].Content.Parts {
		if strings.TrimSpace(part.Text) != "" {
			answerParts = append(answerParts, strings.TrimSpace(part.Text))
		}
	}
	answer := strings.TrimSpace(strings.Join(answerParts, " "))
	if answer == "" {
		return "", errors.New("Gemini returned empty text")
	}

	updated := append(contents, geminiContent{Role: "model", Parts: []geminiPart{{Text: answer}}})
	if len(updated) > chatbotHistoryLimit {
		updated = updated[len(updated)-chatbotHistoryLimit:]
	}
	chatbotHistoryMu.Lock()
	if len(chatbotHistory) >= chatbotHistoryCap {
		for oldKey := range chatbotHistory {
			delete(chatbotHistory, oldKey)
			delete(chatbotLanguages, oldKey)
			break
		}
	}
	chatbotHistory[key] = updated
	chatbotHistoryMu.Unlock()
	return answer, nil
}

func (moduleStruct) chatbotToggle(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg == nil || msg.From == nil {
		return ext.EndGroups
	}
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		_, _ = msg.Reply(b, "Use /chatbot enable or /chatbot disable in a group.", nil)
		return ext.EndGroups
	}
	args := ctx.Args()
	if len(args) != 2 || (strings.ToLower(args[1]) != "enable" && strings.ToLower(args[1]) != "disable") {
		_, _ = msg.Reply(b, "Usage: /chatbot enable|disable", nil)
		return ext.EndGroups
	}
	if !chat_status.IsUserAdmin(b, msg.Chat.Id, msg.From.Id) {
		_, _ = msg.Reply(b, "Only group admins can change the chatbot setting.", nil)
		return ext.EndGroups
	}
	enabled := strings.EqualFold(args[1], "enable")
	if err := chatbotdb.SetEnabled(msg.Chat.Id, enabled); err != nil {
		log.Errorf("[ChatBot] failed to update chat %d setting: %v", msg.Chat.Id, err)
		_, _ = msg.Reply(b, "Couldn't update chatbot setting. Please try again.", nil)
		return ext.EndGroups
	}
	if enabled {
		_, _ = msg.Reply(b, "🦊 Group chatbot enabled! Reply to my message or say Foxy to chat.", nil)
	} else {
		_, _ = msg.Reply(b, "🦊 Group chatbot disabled.", nil)
	}
	return ext.EndGroups
}

func (moduleStruct) chatbotMessage(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg == nil || msg.From == nil || strings.TrimSpace(msg.Text) == "" || strings.HasPrefix(msg.Text, "/") {
		return ext.ContinueGroups
	}
	if msg.Chat.Type == "private" {
		// Private chats respond directly, matching the supplied Foxy implementation.
	} else {
		if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
			return ext.ContinueGroups
		}
		if !chatbotdb.IsEnabled(msg.Chat.Id) {
			return ext.ContinueGroups
		}
		// In groups, either reply to Alita's message (Foxy-style) or say
		// the standalone name "Foxy" anywhere in the message.
		repliedToBot := msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil && msg.ReplyToMessage.From.Id == b.Id
		calledFoxy := chatbotFoxyTrigger.MatchString(msg.Text)
		if !repliedToBot && !calledFoxy {
			return ext.ContinueGroups
		}
	}
	_, _ = b.SendChatAction(msg.Chat.Id, "typing", nil)
	answer, err := chatbotGenerateReply(msg.Chat.Id, msg.From.Id, msg.Text)
	if err != nil {
		log.Errorf("[ChatBot] response generation failed for chat %d: %v", msg.Chat.Id, err)
		if strings.Contains(err.Error(), "not configured") {
			_, _ = msg.Reply(b, answer, nil)
		} else {
			_, _ = msg.Reply(b, "Aww, meri AI abhi reply nahi kar paa rahi. Thodi der baad try karo. 💗", nil)
		}
		return ext.ContinueGroups
	}
	if runes := []rune(answer); len(runes) > 4000 {
		answer = string(runes[:4000])
	}
	_, sendErr := msg.Reply(b, answer, nil)
	if sendErr != nil {
		return sendErr
	}
	return ext.ContinueGroups
}

func LoadChatBot(dispatcher *ext.Dispatcher) {
	SetModuleEnabled("ChatBot", true)
	dispatcher.AddHandler(handlers.NewCommand("chatbot", (moduleStruct{}).chatbotToggle))
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.All, (moduleStruct{}).chatbotMessage), chatbotHandlerGroup)
}

func init() { RegisterLegacyModule("ChatBot", 15, LoadChatBot) }
