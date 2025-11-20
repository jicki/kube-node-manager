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
	hasPassword := false
	if dbConfig.Password != "" {
		dsn += fmt.Sprintf(" password=%s", dbConfig.Password)
		hasPassword = true
	}
	
	logger.Infof("Initializing PostgreSQL listener with host=%s port=%d dbname=%s sslmode=%s password_set=%v", 
		dbConfig.Host, dbConfig.Port, dbConfig.Database, dbConfig.SSLMode, hasPassword)
	
	// 首先验证 GORM 数据库连接是否可用
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("main database connection is unhealthy, cannot create listener: %w (host=%s port=%d)", 
			err, dbConfig.Host, dbConfig.Port)
	}
	logger.Debugf("Main database connection verified, proceeding with listener setup")
	
	// 创建 PostgreSQL Listener
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			// 记录所有带错误的事件
			logger.Errorf("PostgreSQL listener event [%s]: %v", ev, err)
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
			logger.Errorf("PostgreSQL listener connection failed")
			logger.Errorf("  Host: %s:%d", dbConfig.Host, dbConfig.Port)
			logger.Errorf("  Database: %s", dbConfig.Database)
			logger.Errorf("  Username: %s", dbConfig.Username)
			logger.Errorf("  SSL Mode: %s", dbConfig.SSLMode)
			logger.Errorf("  Password set: %v", hasPassword)
			logger.Errorf("  Error: %v", err)
			logger.Errorf("Please check:")
			logger.Errorf("  1. Network connectivity: can pods reach %s:%d?", dbConfig.Host, dbConfig.Port)
			logger.Errorf("  2. Database credentials are correct")
			logger.Errorf("  3. PostgreSQL server is running and accepting connections")
			logger.Errorf("  4. Firewall rules allow connections from pods")
			return nil, fmt.Errorf("failed to connect PostgreSQL listener: %w (host=%s port=%d)", 
				err, dbConfig.Host, dbConfig.Port)
		}
		logger.Infof("✅ PostgreSQL listener connected successfully (verified via ping)")
	case <-ctx.Done():
		listener.Close()
		logger.Errorf("PostgreSQL listener connection timeout after 10s (host=%s port=%d)", 
			dbConfig.Host, dbConfig.Port)
		return nil, fmt.Errorf("PostgreSQL listener connection timeout after 10s (host=%s port=%d)", 
			dbConfig.Host, dbConfig.Port)
	}
	
	notifier := &PostgresNotifier{
		db:       db,
		logger:   logger,
		listener: listener,
	}
	
	logger.Infof("✅ PostgreSQL LISTEN/NOTIFY notifier initialized successfully")
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
	
	// 只记录重要消息（complete, error），避免日志轰炸
	if message.Type == "complete" || message.Type == "error" {
		p.logger.Infof("Sending PostgreSQL notification: task=%s type=%s user=%d", 
			message.TaskID, message.Type, message.UserID)
	}
	
	// 先检查 GORM DB 连接状态
	sqlDB, err := p.db.DB()
	if err != nil {
		p.logger.Errorf("Failed to get underlying sql.DB: %v", err)
		return fmt.Errorf("failed to get database: %w", err)
	}
	
	// 检查连接是否健康
	if err := sqlDB.Ping(); err != nil {
		p.logger.Errorf("Database connection unhealthy: %v", err)
		return fmt.Errorf("database connection error: %w", err)
	}
	
	// 使用原生 SQL 执行 pg_notify（使用 $1, $2 占位符而不是 ?）
	result := p.db.WithContext(ctx).Exec("SELECT pg_notify($1, $2)", channel, string(payload))
	
	if result.Error != nil {
		p.logger.Errorf("Failed to send PostgreSQL notification: %v", result.Error)
		return fmt.Errorf("failed to notify: %w", result.Error)
	}
	
	// 只记录重要消息的成功发送
	if message.Type == "complete" || message.Type == "error" {
		p.logger.Debugf("PostgreSQL notification sent successfully (payload: %d bytes)", len(payload))
	}
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
					continue
				}
				
				var msg ProgressMessage
				if err := json.Unmarshal([]byte(notification.Extra), &msg); err != nil {
					p.logger.Errorf("Failed to unmarshal notification payload: %v", err)
					continue
				}
				
				// 只记录重要消息（complete, error）
				if msg.Type == "complete" || msg.Type == "error" {
					p.logger.Infof("Received PostgreSQL notification: task=%s type=%s user=%d", 
						msg.TaskID, msg.Type, msg.UserID)
				}
				
				select {
				case messageChan <- msg:
					// 只记录重要消息的转发
					if msg.Type == "complete" || msg.Type == "error" {
						p.logger.Debugf("Forwarded %s notification for task %s to WebSocket", 
							msg.Type, msg.TaskID)
					}
				case <-ctx.Done():
					p.logger.Info("Context cancelled while forwarding message")
					return
				default:
					p.logger.Warningf("Message channel full, dropping notification for task %s", msg.TaskID)
				}
				
			case <-time.After(90 * time.Second):
				// 定期 ping 以保持连接（不记录成功日志，避免日志轰炸）
				go func() {
					if err := p.listener.Ping(); err != nil {
						p.logger.Warningf("PostgreSQL listener ping failed: %v", err)
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

