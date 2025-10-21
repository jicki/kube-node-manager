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

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
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

// FeishuUserInfoResponse 飞书用户信息响应
type FeishuUserInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		User struct {
			OpenID          string   `json:"open_id"`
			UnionID         string   `json:"union_id"`
			UserID          string   `json:"user_id"`
			Name            string   `json:"name"`
			EnName          string   `json:"en_name"`
			Email           string   `json:"email"`
			Mobile          string   `json:"mobile"`
			Gender          int      `json:"gender"`
			Avatar          Avatar   `json:"avatar"`
			Status          Status   `json:"status"`
			DepartmentIDs   []string `json:"department_ids"`
			LeaderUserID    string   `json:"leader_user_id"`
			City            string   `json:"city"`
			Country         string   `json:"country"`
			WorkStation     string   `json:"work_station"`
			JoinTime        int64    `json:"join_time"`
			IsTenantManager bool     `json:"is_tenant_manager"`
			EmployeeNo      string   `json:"employee_no"`
			EmployeeType    int      `json:"employee_type"`
		} `json:"user"`
	} `json:"data"`
}

// Avatar 飞书用户头像
type Avatar struct {
	Avatar72     string `json:"avatar_72"`
	Avatar240    string `json:"avatar_240"`
	Avatar640    string `json:"avatar_640"`
	AvatarOrigin string `json:"avatar_origin"`
}

// Status 飞书用户状态
type Status struct {
	IsFrozen    bool `json:"is_frozen"`
	IsResigned  bool `json:"is_resigned"`
	IsActivated bool `json:"is_activated"`
}

// GetFeishuUserInfo 从飞书 API 获取用户信息
func (s *Service) GetFeishuUserInfo(openID string) (*FeishuUserInfoResponse, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// 获取 access token
	token, err := s.getTenantAccessToken(settings.AppID, settings.AppSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// 调用飞书 API 获取用户信息
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=open_id", openID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var userInfo FeishuUserInfoResponse
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if userInfo.Code != 0 {
		return nil, fmt.Errorf("feishu API error: code=%d, msg=%s", userInfo.Code, userInfo.Msg)
	}

	return &userInfo, nil
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

// AutoMatchAndBindUser 自动匹配并绑定飞书用户到系统用户
func (s *Service) AutoMatchAndBindUser(openID string) (*model.FeishuUserMapping, error) {
	s.logger.Info(fmt.Sprintf("🔄 尝试自动匹配用户，Open ID: %s", openID))

	// 1. 获取飞书用户信息
	feishuUserInfo, err := s.GetFeishuUserInfo(openID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 获取飞书用户信息失败: %s", err.Error()))
		return nil, fmt.Errorf("获取飞书用户信息失败: %w", err)
	}

	email := feishuUserInfo.Data.User.Email
	name := feishuUserInfo.Data.User.Name

	s.logger.Info(fmt.Sprintf("📧 飞书用户信息 - 姓名: %s, 邮箱: %s", name, email))

	// 2. 尝试通过邮箱匹配系统用户
	var systemUser model.User
	if email != "" {
		err = s.db.Where("email = ?", email).First(&systemUser).Error
		if err == nil {
			s.logger.Info(fmt.Sprintf("✅ 通过邮箱匹配到系统用户: %s (ID: %d)", systemUser.Username, systemUser.ID))

			// 3. 创建绑定关系
			mapping := &model.FeishuUserMapping{
				FeishuUserID: openID,
				SystemUserID: systemUser.ID,
				Username:     systemUser.Username,
				FeishuName:   name,
			}

			if err := s.db.Create(mapping).Error; err != nil {
				s.logger.Error(fmt.Sprintf("❌ 创建绑定关系失败: %s", err.Error()))
				return nil, fmt.Errorf("创建绑定关系失败: %w", err)
			}

			// 预加载用户信息
			if err := s.db.Preload("User").First(mapping, mapping.ID).Error; err != nil {
				return nil, err
			}

			s.logger.Info(fmt.Sprintf("✅ 自动绑定成功！Feishu: %s -> System: %s", name, systemUser.Username))
			return mapping, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error(fmt.Sprintf("❌ 查询系统用户失败: %s", err.Error()))
			return nil, fmt.Errorf("查询系统用户失败: %w", err)
		}
	}

	// 4. 如果邮箱匹配失败，尝试通过用户名匹配（如果飞书用户名和系统用户名一致）
	if name != "" {
		err = s.db.Where("username = ?", name).First(&systemUser).Error
		if err == nil {
			s.logger.Info(fmt.Sprintf("✅ 通过用户名匹配到系统用户: %s (ID: %d)", systemUser.Username, systemUser.ID))

			// 创建绑定关系
			mapping := &model.FeishuUserMapping{
				FeishuUserID: openID,
				SystemUserID: systemUser.ID,
				Username:     systemUser.Username,
				FeishuName:   name,
			}

			if err := s.db.Create(mapping).Error; err != nil {
				s.logger.Error(fmt.Sprintf("❌ 创建绑定关系失败: %s", err.Error()))
				return nil, fmt.Errorf("创建绑定关系失败: %w", err)
			}

			// 预加载用户信息
			if err := s.db.Preload("User").First(mapping, mapping.ID).Error; err != nil {
				return nil, err
			}

			s.logger.Info(fmt.Sprintf("✅ 自动绑定成功！Feishu: %s -> System: %s", name, systemUser.Username))
			return mapping, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error(fmt.Sprintf("❌ 查询系统用户失败: %s", err.Error()))
			return nil, fmt.Errorf("查询系统用户失败: %w", err)
		}
	}

	// 5. 如果都匹配失败，返回 nil
	s.logger.Info(fmt.Sprintf("⚠️ 无法自动匹配用户 - 飞书姓名: %s, 邮箱: %s", name, email))
	return nil, nil
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

	// 只支持单聊（p2p），不支持群聊 - 直接忽略，不回复
	if chatType != "p2p" {
		s.logger.Info(fmt.Sprintf("⚠️ 机器人只支持单聊，忽略群聊消息。Chat Type: %s", chatType))
		return nil
	}

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
		errorMsg := BuildErrorCard("❌ 没有权限操作\n\n请联系管理员。")
		s.SendMessage(chatID, "interactive", errorMsg)
		return nil
	}

	// 如果用户未绑定，尝试自动匹配
	if userMapping == nil {
		s.logger.Info(fmt.Sprintf("⚠️ 用户未绑定系统账号，Feishu User ID: %s", senderID))
		s.logger.Info("🔄 尝试自动匹配并绑定用户...")

		// 尝试自动匹配
		userMapping, err = s.AutoMatchAndBindUser(senderID)
		if err != nil {
			s.logger.Error(fmt.Sprintf("❌ 自动匹配用户失败: %s", err.Error()))
			errorMsg := BuildErrorCard("❌ 没有权限操作\n\n请联系管理员。")
			s.SendMessage(chatID, "interactive", errorMsg)
			return nil
		}

		// 如果自动匹配也失败，提示用户
		if userMapping == nil {
			s.logger.Info(fmt.Sprintf("⚠️ 无法自动匹配用户"))
			errorMsg := BuildErrorCard("❌ 没有权限操作\n\n请联系管理员。")
			s.logger.Info("📤 准备发送权限错误提示消息...")
			sendErr := s.SendMessage(chatID, "interactive", errorMsg)
			if sendErr != nil {
				s.logger.Error(fmt.Sprintf("❌ 发送提示消息失败: %s", sendErr.Error()))
			} else {
				s.logger.Info("✅ 已成功发送提示消息")
			}
			return nil
		}

		// 自动匹配成功，发送欢迎消息
		s.logger.Info(fmt.Sprintf("🎉 自动匹配成功！"))
		welcomeMsg := BuildSuccessCard(fmt.Sprintf("✅ 账号绑定成功！\n\n"+
			"欢迎使用 Kube 管理机器人！\n\n"+
			"系统账号: %s\n"+
			"角色: %s\n\n"+
			"输入 /help 查看可用命令。", userMapping.Username, userMapping.User.Role))
		s.SendMessage(chatID, "interactive", welcomeMsg)
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

// handleCardAction handles card button click events
func (s *Service) handleCardAction(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	s.logger.Info("========== 收到飞书卡片交互事件 ==========")

	// 提取事件数据
	actionValue := event.Event.Action.Value
	actionValueJSON, err := json.Marshal(actionValue)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 序列化 action value 失败: %s", err.Error()))
		return nil, err
	}
	actionValueStr := string(actionValueJSON)
	s.logger.Info(fmt.Sprintf("📋 Action Value: %s", actionValueStr))

	// 获取操作者信息
	operatorID := event.Event.Operator.OpenID
	s.logger.Info(fmt.Sprintf("👤 操作者 ID: %s", operatorID))

	if operatorID == "" {
		s.logger.Error("❌ 无法获取操作者 ID")
		return nil, fmt.Errorf("operator ID not found")
	}

	// 获取用户绑定信息
	s.logger.Info(fmt.Sprintf("🔍 查询用户绑定状态，Feishu User ID: %s", operatorID))
	userMapping, err := s.GetBindingByFeishuUserID(operatorID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 查询用户绑定失败: %s", err.Error()))
		return nil, err
	}

	if userMapping == nil {
		s.logger.Info("⚠️ 用户未绑定，尝试自动匹配...")
		userMapping, err = s.AutoMatchAndBindUser(operatorID)
		if err != nil || userMapping == nil {
			s.logger.Error("❌ 用户未绑定且自动匹配失败")
			return nil, fmt.Errorf("user not bound")
		}
	}

	s.logger.Info(fmt.Sprintf("✅ 用户已绑定，System User ID: %d, Username: %s",
		userMapping.SystemUserID, userMapping.Username))

	// 检查用户权限
	if userMapping.User.Role != model.RoleAdmin {
		s.logger.Info(fmt.Sprintf("⚠️ 用户权限不足，角色: %s", userMapping.User.Role))
		return nil, fmt.Errorf("insufficient permissions")
	}

	// 创建 CardActionHandler 并处理
	s.logger.Info("🎯 准备处理卡片交互...")
	handler := NewCardActionHandler(s)
	response, err := handler.HandleCardAction(actionValueStr, userMapping)
	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 处理卡片交互失败: %s", err.Error()))
		return nil, err
	}

	// 发送响应消息
	chatID := event.Event.Context.OpenChatID
	if chatID == "" {
		s.logger.Error("❌ 无法获取 chat ID")
		return nil, fmt.Errorf("chat ID not found")
	}

	s.logger.Info(fmt.Sprintf("📤 发送响应消息到 Chat ID: %s", chatID))

	if response.Card != "" {
		err = s.SendMessage(chatID, "interactive", response.Card)
	} else if response.Text != "" {
		textContent := map[string]interface{}{
			"text": response.Text,
		}
		textJSON, _ := json.Marshal(textContent)
		err = s.SendMessage(chatID, "text", string(textJSON))
	}

	if err != nil {
		s.logger.Error(fmt.Sprintf("❌ 发送响应消息失败: %s", err.Error()))
		return nil, err
	}

	s.logger.Info("✅ 卡片交互处理完成")
	s.logger.Info("========== 飞书卡片交互处理完成 ==========")

	// 返回空响应表示成功
	return &callback.CardActionTriggerResponse{}, nil
}
