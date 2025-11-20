package progress

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"kube-node-manager/internal/config"
	"kube-node-manager/pkg/logger"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ProgressNotifier 进度通知接口
type ProgressNotifier interface {
	// Notify 发送进度通知
	Notify(ctx context.Context, message ProgressMessage) error
	
	// Subscribe 订阅进度通知，返回消息通道
	Subscribe(ctx context.Context) (<-chan ProgressMessage, error)
	
	// Close 关闭通知器
	Close() error
	
	// Type 返回通知器类型
	Type() string
}

// PostgresNotifier PostgreSQL LISTEN/NOTIFY 通知器
type PostgresNotifier struct {
	db       *gorm.DB
	logger   *logger.Logger
	listener *pq.Listener
	cancel   context.CancelFunc
}

// NewPostgresNotifier 创建 PostgreSQL 通知器
func NewPostgresNotifier(db *gorm.DB, dbConfig *config.DatabaseConfig, logger *logger.Logger) (*PostgresNotifier, error) {
	// 从配置构建 DSN（与主应用使用相同的配置）
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Username,
		dbConfig.Database,
		dbConfig.SSLMode,
	)
	
	// 添加密码（如果存在）
	if dbConfig.Password != "" {
		dsn += fmt.Sprintf(" password=%s", dbConfig.Password)
	}
	
	logger.Infof("Initializing PostgreSQL listener with host=%s port=%d dbname=%s", 
		dbConfig.Host, dbConfig.Port, dbConfig.Database)
	
	// 创建 PostgreSQL Listener
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			logger.Errorf("PostgreSQL listener [%s]: %v", ev, err)
		}
	}
	
	listener := pq.NewListener(dsn, 10*time.Second, time.Minute, reportProblem)
	
	// 尝试立即 ping 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	connected := make(chan error, 1)
	go func() {
		connected <- listener.Ping()
	}()
	
	select {
	case err := <-connected:
		if err != nil {
			listener.Close()
			return nil, fmt.Errorf("failed to connect PostgreSQL listener: %w (host=%s port=%d)", 
				err, dbConfig.Host, dbConfig.Port)
		}
		logger.Infof("PostgreSQL listener connected successfully (verified via ping)")
	case <-ctx.Done():
		listener.Close()
		return nil, fmt.Errorf("PostgreSQL listener connection timeout after 10s (host=%s port=%d)", 
			dbConfig.Host, dbConfig.Port)
	}
	
	notifier := &PostgresNotifier{
		db:       db,
		logger:   logger,
		listener: listener,
	}
	
	logger.Info("PostgreSQL LISTEN/NOTIFY notifier initialized successfully")
	return notifier, nil
}

// Notify 发送通知
func (p *PostgresNotifier) Notify(ctx context.Context, message ProgressMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// 使用 pg_notify 发送通知
	channel := "progress_update"  // 使用固定通道名
	
	// 记录发送详情（用于调试）
	p.logger.Debugf("Publishing to PostgreSQL channel '%s': task=%s type=%s user=%d success=%d failed=%d", 
		channel, message.TaskID, message.Type, message.UserID, 
		len(message.SuccessNodes), len(message.FailedNodes))
	
	result := p.db.Exec("SELECT pg_notify(?, ?)", channel, string(payload))
	
	if result.Error != nil {
		p.logger.Errorf("Failed to send PostgreSQL notification: %v", result.Error)
		return fmt.Errorf("failed to notify: %w", result.Error)
	}
	
	p.logger.Debugf("Successfully sent PostgreSQL notification to channel '%s' (payload size: %d bytes)", 
		channel, len(payload))
	return nil
}

// Subscribe 订阅通知
func (p *PostgresNotifier) Subscribe(ctx context.Context) (<-chan ProgressMessage, error) {
	// 监听 progress_update 通道
	if err := p.listener.Listen("progress_update"); err != nil {
		p.logger.Errorf("Failed to listen on channel 'progress_update': %v", err)
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	
	messageChan := make(chan ProgressMessage, 100)
	
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	
	go func() {
		defer close(messageChan)
		p.logger.Info("PostgreSQL notification subscription loop started")
		
		for {
			select {
			case <-ctx.Done():
				p.logger.Info("PostgreSQL notifier subscription stopped (context cancelled)")
				return
				
			case notification := <-p.listener.Notify:
				if notification == nil {
					// nil notification can occur during reconnection
					p.logger.Debug("Received nil notification (listener reconnecting)")
					continue
				}
				
				p.logger.Debugf("Received notification on channel '%s': payload size=%d bytes", 
					notification.Channel, len(notification.Extra))
				
				var msg ProgressMessage
				if err := json.Unmarshal([]byte(notification.Extra), &msg); err != nil {
					p.logger.Errorf("Failed to unmarshal notification payload: %v (payload: %s)", 
						err, notification.Extra[:min(100, len(notification.Extra))])
					continue
				}
				
				p.logger.Debugf("Parsed notification: task=%s type=%s user=%d success=%d failed=%d", 
					msg.TaskID, msg.Type, msg.UserID, len(msg.SuccessNodes), len(msg.FailedNodes))
				
				select {
				case messageChan <- msg:
					p.logger.Debugf("Successfully forwarded notification for task %s (type=%s) to WebSocket", 
						msg.TaskID, msg.Type)
				case <-ctx.Done():
					p.logger.Info("Context cancelled while forwarding message")
					return
				default:
					p.logger.Warningf("Message channel full, dropping notification for task %s", msg.TaskID)
				}
				
			case <-time.After(90 * time.Second):
				// 定期 ping 以保持连接
				go func() {
					if err := p.listener.Ping(); err != nil {
						p.logger.Warningf("PostgreSQL listener ping failed (connection may be lost): %v", err)
					} else {
						p.logger.Debug("PostgreSQL listener ping successful")
					}
				}()
			}
		}
	}()
	
	p.logger.Info("PostgreSQL notifier subscription started successfully")
	return messageChan, nil
}

// Close 关闭通知器
func (p *PostgresNotifier) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}

// Type 返回通知器类型
func (p *PostgresNotifier) Type() string {
	return "postgres"
}

// RedisNotifier Redis Pub/Sub 通知器
type RedisNotifier struct {
	client *redis.Client
	logger *logger.Logger
	pubsub *redis.PubSub
	cancel context.CancelFunc
}

// NewRedisNotifier 创建 Redis 通知器
func NewRedisNotifier(addr, password string, db int, logger *logger.Logger) (*RedisNotifier, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	notifier := &RedisNotifier{
		client: client,
		logger: logger,
	}
	
	logger.Infof("Redis Pub/Sub notifier initialized (addr: %s, db: %d)", addr, db)
	return notifier, nil
}

// Notify 发送通知
func (r *RedisNotifier) Notify(ctx context.Context, message ProgressMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// 发布到 Redis 频道
	channel := fmt.Sprintf("progress:user:%d", message.UserID)
	if err := r.client.Publish(ctx, channel, payload).Err(); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}
	
	r.logger.Debugf("Published to Redis channel %s", channel)
	return nil
}

// Subscribe 订阅通知
func (r *RedisNotifier) Subscribe(ctx context.Context) (<-chan ProgressMessage, error) {
	// 订阅所有用户的进度通道（使用模式订阅）
	r.pubsub = r.client.PSubscribe(ctx, "progress:user:*")
	
	messageChan := make(chan ProgressMessage, 100)
	
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	
	go func() {
		defer close(messageChan)
		
		ch := r.pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				r.logger.Info("Redis notifier subscription stopped")
				return
				
			case msg := <-ch:
				if msg == nil {
					continue
				}
				
				var progressMsg ProgressMessage
				if err := json.Unmarshal([]byte(msg.Payload), &progressMsg); err != nil {
					r.logger.Errorf("Failed to unmarshal Redis message: %v", err)
					continue
				}
				
				select {
				case messageChan <- progressMsg:
					r.logger.Debugf("Forwarded Redis notification for task %s", progressMsg.TaskID)
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	r.logger.Info("Redis notifier subscription started")
	return messageChan, nil
}

// Close 关闭通知器
func (r *RedisNotifier) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
	if r.pubsub != nil {
		return r.pubsub.Close()
	}
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// Type 返回通知器类型
func (r *RedisNotifier) Type() string {
	return "redis"
}

// PollingNotifier 轮询模式通知器（回退方案）
type PollingNotifier struct {
	logger       *logger.Logger
	pollInterval time.Duration
}

// NewPollingNotifier 创建轮询通知器
func NewPollingNotifier(pollInterval time.Duration, logger *logger.Logger) *PollingNotifier {
	return &PollingNotifier{
		logger:       logger,
		pollInterval: pollInterval,
	}
}

// Notify 轮询模式不主动发送通知（通过轮询获取）
func (p *PollingNotifier) Notify(ctx context.Context, message ProgressMessage) error {
	// 轮询模式依赖数据库轮询，这里不做任何操作
	return nil
}

// Subscribe 轮询模式不支持订阅
func (p *PollingNotifier) Subscribe(ctx context.Context) (<-chan ProgressMessage, error) {
	return nil, fmt.Errorf("polling notifier does not support subscription")
}

// Close 关闭通知器
func (p *PollingNotifier) Close() error {
	return nil
}

// Type 返回通知器类型
func (p *PollingNotifier) Type() string {
	return "polling"
}

// getEnvOrDefault 获取环境变量或返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

