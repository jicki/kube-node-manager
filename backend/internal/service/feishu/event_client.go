package feishu

import (
	"context"
	"fmt"
	"sync"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

// EventClient 管理飞书长连接客户端
type EventClient struct {
	service   *Service
	appID     string
	appSecret string
	client    *lark.Client
	wsClient  *larkws.Client
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	connected bool
}

// NewEventClient 创建新的事件客户端
func NewEventClient(service *Service, appID, appSecret string) *EventClient {
	eventCtx, cancel := context.WithCancel(context.Background())

	// 创建飞书客户端
	client := lark.NewClient(appID, appSecret)

	return &EventClient{
		service:   service,
		appID:     appID,
		appSecret: appSecret,
		client:    client,
		ctx:       eventCtx,
		cancel:    cancel,
		connected: false,
	}
}

// Start 启动长连接
func (ec *EventClient) Start() error {
	ec.mu.Lock()
	if ec.connected {
		ec.mu.Unlock()
		return fmt.Errorf("event client already started")
	}
	ec.mu.Unlock()

	ec.service.logger.Info("Starting Feishu event client with long connection...")

	// 创建事件分发器
	handler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			return ec.service.handleMessageReceive(ctx, event)
		})

	// 创建 WebSocket 客户端
	cli := larkws.NewClient(ec.appID, ec.appSecret,
		larkws.WithEventHandler(handler),
	)

	ec.mu.Lock()
	ec.wsClient = cli
	ec.mu.Unlock()

	// 使用 channel 来等待连接建立
	connectedChan := make(chan bool, 1)

	// 启动长连接（异步）
	go func() {
		ec.service.logger.Info("Feishu WebSocket client starting...")

		// 设置连接状态为 true（SDK 会在后台建立连接）
		ec.mu.Lock()
		ec.connected = true
		ec.mu.Unlock()
		connectedChan <- true

		// Start 是阻塞的，会一直运行直到连接断开
		err := cli.Start(ec.ctx)
		if err != nil {
			ec.service.logger.Error("Feishu event client stopped with error: " + err.Error())
		} else {
			ec.service.logger.Info("Feishu event client stopped normally")
		}

		ec.mu.Lock()
		ec.connected = false
		ec.mu.Unlock()
	}()

	// 等待连接启动（最多等待 5 秒）
	select {
	case <-connectedChan:
		ec.service.logger.Info("Feishu event client connection initialized")
		return nil
	case <-ec.ctx.Done():
		return fmt.Errorf("context cancelled")
	}
}

// Stop 停止长连接
func (ec *EventClient) Stop() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if !ec.connected {
		return nil
	}

	ec.service.logger.Info("Stopping Feishu event client...")

	// 取消上下文
	ec.cancel()
	ec.connected = false

	ec.service.logger.Info("Feishu event client stopped")

	return nil
}

// IsConnected 检查连接状态
func (ec *EventClient) IsConnected() bool {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.connected
}
