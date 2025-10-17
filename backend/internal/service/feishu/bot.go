package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kube-node-manager/internal/model"
	"net/http"
	"strings"
	"time"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"gorm.io/gorm"
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
	// 预加载用户信息以便进行权限检查
	if err := s.db.Preload("User").Where("feishu_user_id = ?", feishuUserID).First(&mapping).Error; err != nil {
		// 如果是记录不存在，返回 nil, nil（表示用户未绑定）
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		// 其他错误才返回错误
		return nil, err
	}
	return &mapping, nil
}

// SendMessage sends a message to a chat
func (s *Service) SendMessage(chatID, msgType, content string) error {
	s.logger.Info("📨 ========== 开始发送飞书消息 ==========")
	s.logger.Info(fmt.Sprintf("Chat ID: %s", chatID))
	s.logger.Info(fmt.Sprintf("消息类型: %s", msgType))
	s.logger.Info(fmt.Sprintf("消息内容长度: %d 字节", len(content)))

	settings, err := s.GetSettings()
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 获取飞书配置失败: %s", err.Error()))
		return fmt.Errorf("failed to get settings: %w", err)
	}
	s.logger.Info(fmt.Sprintf("✅ 已获取飞书配置，App ID: %s", settings.AppID))

	// Get access token
	s.logger.Info("🔑 正在获取 Access Token...")
	token, err := s.getTenantAccessToken(settings.AppID, settings.AppSecret)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 获取 Access Token 失败: %s", err.Error()))
		return err
	}
	s.logger.Info(fmt.Sprintf("✅ Access Token 获取成功，长度: %d", len(token)))

	// Prepare request
	url := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id"

	reqBody := SendMessageRequest{
		ReceiveID: chatID,
		MsgType:   msgType,
		Content:   content,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 序列化请求体失败: %s", err.Error()))
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	s.logger.Info(fmt.Sprintf("✅ 请求体已准备，大小: %d 字节", len(jsonData)))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 创建 HTTP 请求失败: %s", err.Error()))
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	s.logger.Info(fmt.Sprintf("🌐 正在发送 HTTP 请求到: %s", url))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ HTTP 请求失败: %s", err.Error()))
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	s.logger.Info(fmt.Sprintf("✅ 收到 HTTP 响应，状态码: %d", resp.StatusCode))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 读取响应体失败: %s", err.Error()))
		return fmt.Errorf("failed to read response: %w", err)
	}
	s.logger.Info(fmt.Sprintf("响应体内容: %s", string(body)))

	var sendResp SendMessageResponse
	if err := json.Unmarshal(body, &sendResp); err != nil {
		s.logger.Error(fmt.Sprintf("❌ 解析响应失败: %s", err.Error()))
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if sendResp.Code != 0 {
		s.logger.Error(fmt.Sprintf("❌ 飞书 API 返回错误: code=%d, msg=%s", sendResp.Code, sendResp.Msg))
		return fmt.Errorf("feishu API error: code=%d, msg=%s", sendResp.Code, sendResp.Msg)
	}

	s.logger.Info(fmt.Sprintf("✅ 消息发送成功！Message ID: %s", sendResp.Data.MessageID))
	s.logger.Info("📨 ========== 飞书消息发送完成 ==========")
	return nil
}

// executeCommand executes a bot command
func (s *Service) executeCommand(command string, userMapping *model.FeishuUserMapping, chatID, messageID string) {
	s.logger.Info(fmt.Sprintf("---------- 开始执行命令 ----------"))
	s.logger.Info(fmt.Sprintf("命令: %s", command))
	s.logger.Info(fmt.Sprintf("用户: %s (ID: %d)", userMapping.Username, userMapping.SystemUserID))
	s.logger.Info(fmt.Sprintf("Chat ID: %s", chatID))

	// Parse command
	cmd := ParseCommand(command)
	if cmd == nil {
		// Invalid command
		s.logger.Error(fmt.Sprintf("❌ 命令解析失败，无效的命令格式: %s", command))
		errorMsg := BuildErrorCard("无效的命令格式。输入 /help 查看帮助信息。")
		s.SendMessage(chatID, "interactive", errorMsg)
		return
	}

	s.logger.Info(fmt.Sprintf("✅ 命令解析成功 - 名称: %s, 动作: %s, 参数: %v", cmd.Name, cmd.Action, cmd.Args))

	// Execute command through command router
	ctx := &CommandContext{
		Command:     cmd,
		UserMapping: userMapping,
		ChatID:      chatID,
		MessageID:   messageID,
		Service:     s,
	}

	s.logger.Info(fmt.Sprintf("🔄 正在通过命令路由器执行命令..."))
	response, err := s.commandRouter.Route(ctx)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 命令执行失败: %s", err.Error()))
		errorMsg := BuildErrorCard(fmt.Sprintf("命令执行失败：%s", err.Error()))
		s.SendMessage(chatID, "interactive", errorMsg)
		return
	}

	s.logger.Info(fmt.Sprintf("✅ 命令执行成功"))

	// Send response
	if response.Card != "" {
		s.logger.Info(fmt.Sprintf("📤 准备发送交互卡片响应，长度: %d", len(response.Card)))
		err := s.SendMessage(chatID, "interactive", response.Card)
		if err != nil {
			s.logger.Error(fmt.Sprintf("❌ 发送卡片响应失败: %s", err.Error()))
		} else {
			s.logger.Info("✅ 卡片响应发送成功")
		}
	} else if response.Text != "" {
		s.logger.Info(fmt.Sprintf("📤 准备发送文本响应: %s", response.Text))
		content := map[string]interface{}{
			"text": response.Text,
		}
		contentJSON, _ := json.Marshal(content)
		err := s.SendMessage(chatID, MessageTypeText, string(contentJSON))
		if err != nil {
			s.logger.Error(fmt.Sprintf("❌ 发送文本响应失败: %s", err.Error()))
		} else {
			s.logger.Info("✅ 文本响应发送成功")
		}
	} else {
		s.logger.Info("⚠️ 命令执行成功但没有返回任何响应内容")
	}

	s.logger.Info(fmt.Sprintf("---------- 命令执行完成 ----------"))
}

// handleMessageReceive 处理从 SDK 长连接接收到的消息
func (s *Service) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	s.logger.Info("========== 飞书消息接收开始 ==========")
	s.logger.Info(fmt.Sprintf("收到飞书 SDK 消息，Event Type: %v", event.Event))

	// 获取并记录 Chat Type
	chatType := ""
	if event.Event.Message.ChatType != nil {
		chatType = *event.Event.Message.ChatType
	}
	s.logger.Info(fmt.Sprintf("Chat Type: %s", chatType))

	// 获取消息内容
	messageType := event.Event.Message.MessageType
	s.logger.Info(fmt.Sprintf("消息类型: %v", messageType))

	if messageType == nil || *messageType != "text" {
		s.logger.Info(fmt.Sprintf("忽略非文本消息，消息类型: %v", messageType))
		return nil
	}

	// 解析文本内容
	var textContent struct {
		Text string `json:"text"`
	}
	s.logger.Info(fmt.Sprintf("原始消息内容: %s", *event.Event.Message.Content))

	if err := json.Unmarshal([]byte(*event.Event.Message.Content), &textContent); err != nil {
		s.logger.Error(fmt.Sprintf("解析消息内容失败: %s", err.Error()))
		return err
	}

	messageText := strings.TrimSpace(textContent.Text)
	s.logger.Info(fmt.Sprintf("✅ 解析后的消息文本: '%s'", messageText))

	// 记录 mentions 信息
	if len(event.Event.Message.Mentions) > 0 {
		s.logger.Info(fmt.Sprintf("📢 消息包含 %d 个 @提及", len(event.Event.Message.Mentions)))
		for i, mention := range event.Event.Message.Mentions {
			mentionName := ""
			if mention.Name != nil {
				mentionName = *mention.Name
			}
			s.logger.Info(fmt.Sprintf("  [%d] @%s", i+1, mentionName))
		}
	} else {
		s.logger.Info("消息不包含 @提及")
	}

	// 检查是否是命令（以 / 开头）
	if !strings.HasPrefix(messageText, "/") {
		s.logger.Info(fmt.Sprintf("不是命令消息（不以 / 开头），忽略。消息内容: '%s'", messageText))
		return nil
	}

	// 获取发送者 ID
	senderID := ""
	if event.Event.Sender != nil && event.Event.Sender.SenderId != nil {
		if event.Event.Sender.SenderId.OpenId != nil {
			senderID = *event.Event.Sender.SenderId.OpenId
		}
	}
	s.logger.Info(fmt.Sprintf("发送者 Open ID: %s", senderID))

	if senderID == "" {
		s.logger.Error("❌ 无法获取发送者 ID")
		return fmt.Errorf("invalid sender ID")
	}

	// 获取 chat ID
	chatID := ""
	if event.Event.Message.ChatId != nil {
		chatID = *event.Event.Message.ChatId
	}
	s.logger.Info(fmt.Sprintf("Chat ID: %s", chatID))

	if chatID == "" {
		s.logger.Error("❌ 无法获取 Chat ID")
		return fmt.Errorf("invalid chat ID")
	}

	// 获取 message ID
	messageID := ""
	if event.Event.Message.MessageId != nil {
		messageID = *event.Event.Message.MessageId
	}
	s.logger.Info(fmt.Sprintf("Message ID: %s", messageID))

	// 检查用户绑定
	s.logger.Info(fmt.Sprintf("🔍 检查用户绑定状态，Feishu User ID: %s", senderID))
	userMapping, err := s.GetBindingByFeishuUserID(senderID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 查询用户绑定失败: %s", err.Error()))
		errorMsg := BuildErrorCard(fmt.Sprintf("查询绑定状态失败。您的 Open ID: %s", senderID))
		s.SendMessage(chatID, "interactive", errorMsg)
		return nil
	}

	if userMapping == nil {
		s.logger.Info(fmt.Sprintf("⚠️ 用户未绑定系统账号，Feishu User ID: %s", senderID))
		errorMsg := BuildErrorCard(fmt.Sprintf("您尚未绑定系统账号。\n\n您的飞书 Open ID: %s\n\n请在系统中完成账号绑定后再使用机器人功能。", senderID))
		s.logger.Info("📤 准备发送未绑定提示消息...")
		sendErr := s.SendMessage(chatID, "interactive", errorMsg)
		if sendErr != nil {
			s.logger.Error(fmt.Sprintf("❌ 发送未绑定提示消息失败: %s", sendErr.Error()))
		} else {
			s.logger.Info("✅ 已成功发送未绑定提示消息")
		}
		return nil
	}

	s.logger.Info(fmt.Sprintf("✅ 用户已绑定，Feishu User ID: %s -> System User ID: %d, Username: %s",
		senderID, userMapping.SystemUserID, userMapping.Username))

	// 检查用户权限
	s.logger.Info(fmt.Sprintf("🔐 检查用户权限，角色: %s", userMapping.User.Role))
	if userMapping.User.Role != model.RoleAdmin {
		s.logger.Info(fmt.Sprintf("⚠️ 用户权限不足，需要管理员权限。当前角色: %s", userMapping.User.Role))
		errorMsg := BuildErrorCard(fmt.Sprintf("❌ 无权操作\n\n机器人命令仅限管理员使用。\n\n您当前的角色: %s\n请联系管理员申请权限。", userMapping.User.Role))
		s.logger.Info("📤 准备发送权限不足提示消息...")
		sendErr := s.SendMessage(chatID, "interactive", errorMsg)
		if sendErr != nil {
			s.logger.Error(fmt.Sprintf("❌ 发送权限不足提示消息失败: %s", sendErr.Error()))
		} else {
			s.logger.Info("✅ 已成功发送权限不足提示消息")
		}
		return nil
	}
	s.logger.Info("✅ 用户权限验证通过，允许执行命令")

	s.logger.Info(fmt.Sprintf("🚀 准备执行命令: '%s'", messageText))

	// 异步执行命令
	go s.executeCommand(messageText, userMapping, chatID, messageID)

	s.logger.Info("========== 飞书消息接收处理完成 ==========")
	return nil
}
