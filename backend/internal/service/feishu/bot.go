package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kube-node-manager/internal/model"
	"net/http"
	"strings"
	"time"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// Event types
const (
	EventTypeURLVerification = "url_verification"
	EventTypeMessageReceive  = "im.message.receive_v1"
)

// Message types
const (
	MessageTypeText        = "text"
	MessageTypePost        = "post"
	MessageTypeInteractive = "interactive"
)

// FeishuEvent represents a Feishu event callback
type FeishuEvent struct {
	Schema string                 `json:"schema"`
	Header FeishuEventHeader      `json:"header"`
	Event  map[string]interface{} `json:"event"`
}

// FeishuEventHeader represents the event header
type FeishuEventHeader struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	CreateTime string `json:"create_time"`
	Token      string `json:"token"`
	AppID      string `json:"app_id"`
	TenantKey  string `json:"tenant_key"`
}

// URLVerificationEvent represents the URL verification event
type URLVerificationEvent struct {
	Challenge string `json:"challenge"`
	Token     string `json:"token"`
	Type      string `json:"type"`
}

// MessageEvent represents a message received event
type MessageEvent struct {
	Sender    MessageSender  `json:"sender"`
	Message   MessageContent `json:"message"`
	MessageID string         `json:"message_id"`
}

// MessageSender represents the message sender
type MessageSender struct {
	SenderID   SenderID `json:"sender_id"`
	SenderType string   `json:"sender_type"`
	TenantKey  string   `json:"tenant_key"`
}

// SenderID represents the sender ID
type SenderID struct {
	UnionID string `json:"union_id"`
	UserID  string `json:"user_id"`
	OpenID  string `json:"open_id"`
}

// MessageContent represents the message content
type MessageContent struct {
	MessageID   string    `json:"message_id"`
	RootID      string    `json:"root_id"`
	ParentID    string    `json:"parent_id"`
	CreateTime  string    `json:"create_time"`
	ChatID      string    `json:"chat_id"`
	ChatType    string    `json:"chat_type"`
	MessageType string    `json:"message_type"`
	Content     string    `json:"content"`
	Mentions    []Mention `json:"mentions"`
}

// Mention represents a user mention
type Mention struct {
	Key       string   `json:"key"`
	ID        SenderID `json:"id"`
	Name      string   `json:"name"`
	TenantKey string   `json:"tenant_key"`
}

// SendMessageRequest represents a send message request
type SendMessageRequest struct {
	ReceiveID string `json:"receive_id"`
	MsgType   string `json:"msg_type"`
	Content   string `json:"content"`
}

// SendMessageResponse represents the send message response
type SendMessageResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		MessageID string `json:"message_id"`
	} `json:"data"`
}

// GetBindingByFeishuUserID retrieves the user mapping for a Feishu user ID
func (s *Service) GetBindingByFeishuUserID(feishuUserID string) (*model.FeishuUserMapping, error) {
	var mapping model.FeishuUserMapping
	if err := s.db.Where("feishu_user_id = ?", feishuUserID).First(&mapping).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

// SendMessage sends a message to a chat
func (s *Service) SendMessage(chatID, msgType, content string) error {
	settings, err := s.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	// Get access token
	token, err := s.getTenantAccessToken(settings.AppID, settings.AppSecret)
	if err != nil {
		return err
	}

	// Prepare request
	url := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id"

	reqBody := SendMessageRequest{
		ReceiveID: chatID,
		MsgType:   msgType,
		Content:   content,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var sendResp SendMessageResponse
	if err := json.Unmarshal(body, &sendResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if sendResp.Code != 0 {
		return fmt.Errorf("feishu API error: code=%d, msg=%s", sendResp.Code, sendResp.Msg)
	}

	return nil
}

// executeCommand executes a bot command
func (s *Service) executeCommand(command string, userMapping *model.FeishuUserMapping, chatID, messageID string) {
	// Parse command
	cmd := ParseCommand(command)
	if cmd == nil {
		// Invalid command
		errorMsg := BuildErrorCard("无效的命令格式。输入 /help 查看帮助信息。")
		s.SendMessage(chatID, "interactive", errorMsg)
		return
	}

	// Execute command through command router
	ctx := &CommandContext{
		Command:     cmd,
		UserMapping: userMapping,
		ChatID:      chatID,
		MessageID:   messageID,
		Service:     s,
	}

	response, err := s.commandRouter.Route(ctx)
	if err != nil {
		errorMsg := BuildErrorCard(fmt.Sprintf("命令执行失败：%s", err.Error()))
		s.SendMessage(chatID, "interactive", errorMsg)
		return
	}

	// Send response
	if response.Card != "" {
		s.SendMessage(chatID, "interactive", response.Card)
	} else if response.Text != "" {
		content := map[string]interface{}{
			"text": response.Text,
		}
		contentJSON, _ := json.Marshal(content)
		s.SendMessage(chatID, MessageTypeText, string(contentJSON))
	}
}

// handleMessageReceive 处理从 SDK 长连接接收到的消息
func (s *Service) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	s.logger.Info(fmt.Sprintf("Received message from Feishu SDK: %v", event))

	// 获取消息内容
	messageType := event.Event.Message.MessageType
	if messageType == nil || *messageType != "text" {
		s.logger.Info("Ignoring non-text message")
		return nil
	}

	// 解析文本内容
	var textContent struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(*event.Event.Message.Content), &textContent); err != nil {
		s.logger.Error("Failed to parse message content: " + err.Error())
		return err
	}

	messageText := strings.TrimSpace(textContent.Text)
	s.logger.Info(fmt.Sprintf("Message text: %s", messageText))

	// 检查是否是命令（以 / 开头）
	if !strings.HasPrefix(messageText, "/") {
		s.logger.Info("Message is not a command, ignoring")
		return nil
	}

	// 获取发送者 ID
	senderID := ""
	if event.Event.Sender != nil && event.Event.Sender.SenderId != nil {
		if event.Event.Sender.SenderId.OpenId != nil {
			senderID = *event.Event.Sender.SenderId.OpenId
		}
	}

	if senderID == "" {
		s.logger.Error("Failed to get sender ID")
		return fmt.Errorf("invalid sender ID")
	}

	// 获取 chat ID
	chatID := ""
	if event.Event.Message.ChatId != nil {
		chatID = *event.Event.Message.ChatId
	}

	if chatID == "" {
		s.logger.Error("Failed to get chat ID")
		return fmt.Errorf("invalid chat ID")
	}

	// 获取 message ID
	messageID := ""
	if event.Event.Message.MessageId != nil {
		messageID = *event.Event.Message.MessageId
	}

	// 检查用户绑定
	userMapping, err := s.GetBindingByFeishuUserID(senderID)
	if err != nil || userMapping == nil {
		errorMsg := BuildErrorCard("您尚未绑定系统账号，请先绑定后再使用机器人功能。")
		s.SendMessage(chatID, "interactive", errorMsg)
		s.logger.Info(fmt.Sprintf("User %s not bound to system account", senderID))
		return nil
	}

	s.logger.Info(fmt.Sprintf("User %s (system user ID: %d) executing command: %s", senderID, userMapping.SystemUserID, messageText))

	// 异步执行命令
	go s.executeCommand(messageText, userMapping, chatID, messageID)

	return nil
}
