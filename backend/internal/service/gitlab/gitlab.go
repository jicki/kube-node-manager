package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Service handles GitLab-related operations
type Service struct {
	db         *gorm.DB
	logger     *logger.Logger
	httpClient *http.Client
}

// NewService creates a new GitLab service
func NewService(db *gorm.DB, logger *logger.Logger) *Service {
	return &Service{
		db:         db,
		logger:     logger,
		httpClient: createOptimizedHTTPClient(),
	}
}

// createOptimizedHTTPClient 创建一个优化的 HTTP 客户端，具有连接池和超时配置
func createOptimizedHTTPClient() *http.Client {
	transport := &http.Transport{
		// 连接池配置
		MaxIdleConns:        100,              // 最大空闲连接数
		MaxIdleConnsPerHost: 20,               // 每个主机的最大空闲连接数
		MaxConnsPerHost:     50,               // 每个主机的最大连接数
		IdleConnTimeout:     90 * time.Second, // 空闲连接超时时间
		
		// TCP 连接配置
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second, // Keep-alive 探测间隔
		}).DialContext,
		
		// TLS 握手超时
		TLSHandshakeTimeout: 10 * time.Second,
		
		// 响应头超时
		ResponseHeaderTimeout: 15 * time.Second,
		
		// 期望继续超时
		ExpectContinueTimeout: 1 * time.Second,
		
		// 禁用压缩以提高性能（如果 GitLab 响应已压缩）
		DisableCompression: false,
		
		// 禁用 Keep-Alive 会降低性能，保持启用
		DisableKeepAlives: false,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second, // 整体请求超时时间，从 10s 增加到 60s
	}
}

// doHTTPRequestWithRetry 执行 HTTP 请求并支持重试机制
func (s *Service) doHTTPRequestWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：第一次重试等待 100ms，第二次 200ms，第三次 400ms
			backoff := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			
			s.logger.Debug(fmt.Sprintf("Retrying request (attempt %d/%d) after %v: %s", 
				attempt+1, maxRetries+1, backoff, req.URL.String()))
		}
		
		// 使用上下文创建新请求
		reqWithContext := req.WithContext(ctx)
		
		resp, err := s.httpClient.Do(reqWithContext)
		if err != nil {
			lastErr = err
			// 检查是否是可重试的错误
			if isRetriableError(err) {
				continue
			}
			return nil, err
		}
		
		// 检查 HTTP 状态码
		// 5xx 错误和 429 (Rate Limit) 可以重试
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
			
			// 如果是 429，等待更长时间
			if resp.StatusCode == 429 {
				retryAfter := 5 * time.Second
				select {
				case <-time.After(retryAfter):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			continue
		}
		
		// 成功或不可重试的错误（如 4xx）
		return resp, nil
	}
	
	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, lastErr)
}

// isRetriableError 判断错误是否可重试
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	
	// 网络超时错误可重试
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	
	// DNS 解析错误可重试
	if _, ok := err.(*net.DNSError); ok {
		return true
	}
	
	// 连接被拒绝可重试
	if strings.Contains(err.Error(), "connection refused") {
		return true
	}
	
	// EOF 错误可重试
	if strings.Contains(err.Error(), "EOF") {
		return true
	}
	
	return false
}

// SaveRunnerToken saves the runner token to database
func (s *Service) SaveRunnerToken(runnerID int, token, description, runnerType, createdBy string) error {
	s.logger.Info(fmt.Sprintf("Saving token to database for runner_id=%d, created_by=%s", runnerID, createdBy))

	gitlabRunner := model.GitlabRunner{
		RunnerID:    runnerID,
		Token:       token, // In production, this should be encrypted
		Description: description,
		RunnerType:  runnerType,
		CreatedBy:   createdBy,
	}

	if err := s.db.Create(&gitlabRunner).Error; err != nil {
		s.logger.Error(fmt.Sprintf("Failed to save token for runner_id=%d: %v", runnerID, err))
		return err
	}

	s.logger.Info(fmt.Sprintf("Successfully saved token for runner_id=%d", runnerID))
	return nil
}

// GetRunnerToken retrieves the runner token from database
func (s *Service) GetRunnerToken(runnerID int) (*model.GitlabRunner, error) {
	s.logger.Info(fmt.Sprintf("Querying token from database for runner_id=%d", runnerID))

	var runner model.GitlabRunner
	err := s.db.Where("runner_id = ?", runnerID).First(&runner).Error
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to query token for runner_id=%d: %v", runnerID, err))
		return nil, err
	}

	s.logger.Info(fmt.Sprintf("Successfully retrieved token for runner_id=%d, token_length=%d", runnerID, len(runner.Token)))
	return &runner, nil
}

// DeleteRunnerToken deletes the runner token from database
func (s *Service) DeleteRunnerToken(runnerID int) error {
	return s.db.Where("runner_id = ?", runnerID).Delete(&model.GitlabRunner{}).Error
}

// UpdateRunnerToken updates the runner token in database
func (s *Service) UpdateRunnerToken(runnerID int, newToken string) error {
	s.logger.Info(fmt.Sprintf("Updating token in database for runner_id=%d", runnerID))

	result := s.db.Model(&model.GitlabRunner{}).Where("runner_id = ?", runnerID).Update("token", newToken)
	if result.Error != nil {
		s.logger.Error(fmt.Sprintf("Database update error for runner_id=%d: %v", runnerID, result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		s.logger.Warning(fmt.Sprintf("No rows affected when updating runner_id=%d, record may not exist", runnerID))
		return fmt.Errorf("no runner found with runner_id=%d", runnerID)
	}

	s.logger.Info(fmt.Sprintf("Successfully updated token for runner_id=%d, affected rows: %d", runnerID, result.RowsAffected))
	return nil
}

// GetSettings retrieves GitLab settings
func (s *Service) GetSettings() (*model.GitlabSettings, error) {
	var settings model.GitlabSettings

	// Get the first (and should be only) settings record
	if err := s.db.First(&settings).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return default settings if none exist
			return &model.GitlabSettings{
				Enabled: false,
			}, nil
		}
		return nil, err
	}

	return &settings, nil
}

// UpdateSettings updates or creates GitLab settings
func (s *Service) UpdateSettings(enabled bool, domain, token string) (*model.GitlabSettings, error) {
	var settings model.GitlabSettings

	// Try to find existing settings
	err := s.db.First(&settings).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Update fields
	settings.Enabled = enabled
	settings.Domain = strings.TrimRight(domain, "/")

	// Only update token if provided
	if token != "" {
		settings.Token = token
	}

	// Save or create
	if settings.ID == 0 {
		if err := s.db.Create(&settings).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.Save(&settings).Error; err != nil {
			return nil, err
		}
	}

	return &settings, nil
}

// TestConnection tests GitLab API connection
func (s *Service) TestConnection(domain, token string) error {
	if domain == "" || token == "" {
		return errors.New("domain and token are required")
	}

	// Clean domain
	domain = strings.TrimRight(domain, "/")

	// Test API endpoint
	testURL := fmt.Sprintf("%s/api/v4/user", domain)

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", token)

	// 使用带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, req, 2)
	if err != nil {
		return fmt.Errorf("failed to connect to GitLab: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ProjectInfo represents project information for a runner
type ProjectInfo struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	NameWithNamespace string `json:"name_with_namespace"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
}

// GroupInfo represents group information for a runner
type GroupInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
}

// RunnerInfo represents GitLab runner information
type RunnerInfo struct {
	ID           int           `json:"id"`
	Description  string        `json:"description"`
	Active       bool          `json:"active"`
	Paused       bool          `json:"paused"`
	IsShared     bool          `json:"is_shared"`
	IPAddress    string        `json:"ip_address"`
	RunnerType   string        `json:"runner_type"`
	Name         string        `json:"name"`
	Online       bool          `json:"online"`
	Status       string        `json:"status"`
	ContactedAt  *time.Time    `json:"contacted_at"`
	CreatedAt    *time.Time    `json:"created_at"`
	TagList      []string      `json:"tag_list"`
	Version      string        `json:"version"`
	Architecture string        `json:"architecture"`
	Platform     string        `json:"platform"`
	Locked       bool          `json:"locked"`
	AccessLevel  string        `json:"access_level"`
	Projects     []ProjectInfo `json:"projects"`
	Groups       []GroupInfo   `json:"groups"`
}

// UpdateRunnerRequest represents the request to update a runner
type UpdateRunnerRequest struct {
	Description *string   `json:"description,omitempty"`
	Active      *bool     `json:"active,omitempty"`
	TagList     *[]string `json:"tag_list,omitempty"`
	Locked      *bool     `json:"locked,omitempty"`
	AccessLevel *string   `json:"access_level,omitempty"`
}

// CreateRunnerRequest represents the request to create a new runner
type CreateRunnerRequest struct {
	RunnerType     string   `json:"runner_type" binding:"required"` // instance_type, group_type, project_type
	GroupID        *int     `json:"group_id,omitempty"`             // Required for group_type
	ProjectID      *int     `json:"project_id,omitempty"`           // Required for project_type
	Description    string   `json:"description"`
	TagList        []string `json:"tag_list,omitempty"`
	RunUntagged    *bool    `json:"run_untagged,omitempty"`
	Locked         *bool    `json:"locked,omitempty"`
	AccessLevel    *string  `json:"access_level,omitempty"` // not_protected, ref_protected
	MaximumTimeout *int     `json:"maximum_timeout,omitempty"`
	Paused         *bool    `json:"paused,omitempty"`
}

// CreateRunnerResponse represents the response from creating a runner
type CreateRunnerResponse struct {
	ID          int      `json:"id"`
	Token       string   `json:"token"`
	Description string   `json:"description"`
	Active      bool     `json:"active"`
	Paused      bool     `json:"paused"`
	IsShared    bool     `json:"is_shared"`
	RunnerType  string   `json:"runner_type"`
	TagList     []string `json:"tag_list"`
}

// ListRunners retrieves all runners from GitLab
func (s *Service) ListRunners(runnerType string, status string, paused *bool) (interface{}, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if !settings.Enabled {
		return nil, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, errors.New("GitLab domain or token is not configured")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Fetch all runners with pagination
	var allRunners []RunnerInfo
	page := 1
	perPage := 100 // GitLab default max per page

	for {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("operation cancelled or timed out: %w", ctx.Err())
		default:
		}
		
		// Build URL with query parameters
		// Note: /api/v4/runners/all returns basic runner info only
		// Fields like tag_list, contacted_at, version, locked are NOT included
		// To get these fields, use GetRunner(id) for individual runners
		apiURL := fmt.Sprintf("%s/api/v4/runners/all", settings.Domain)
		u, err := url.Parse(apiURL)
		if err != nil {
			return nil, err
		}

		q := u.Query()
		if runnerType != "" {
			q.Set("type", runnerType)
		}
		if status != "" {
			q.Set("status", status)
		}
		if paused != nil {
			if *paused {
				q.Set("paused", "true")
			} else {
				q.Set("paused", "false")
			}
		}
		q.Set("per_page", fmt.Sprintf("%d", perPage))
		q.Set("page", fmt.Sprintf("%d", page))
		u.RawQuery = q.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("PRIVATE-TOKEN", settings.Token)

		// 使用带重试的请求
		resp, err := s.doHTTPRequestWithRetry(ctx, req, 2)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
		}

		// Read body for current page
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var runners []RunnerInfo
		if err := json.Unmarshal(body, &runners); err != nil {
			return nil, err
		}

		// Append runners from current page
		allRunners = append(allRunners, runners...)

		// Check if there are more pages
		// If the current page has fewer runners than per_page, we've reached the last page
		if len(runners) < perPage {
			break
		}

		// Move to next page
		page++
	}

	s.logger.Info(fmt.Sprintf("Fetched total %d runners from GitLab", len(allRunners)))

	// The /runners/all endpoint returns limited information (no tag_list, contacted_at, etc.)
	// Fetch detailed info for each runner concurrently to improve performance
	detailedRunners := make([]RunnerInfo, len(allRunners))

	// Use goroutines with a semaphore to limit concurrent requests
	// Increase concurrency to 20 for better performance (减少并发数以避免过载)
	maxConcurrent := 20
	sem := make(chan struct{}, maxConcurrent)

	// Use sync.WaitGroup for better synchronization
	var wg sync.WaitGroup
	wg.Add(len(allRunners))

	for i, runner := range allRunners {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			// 如果超时，使用基本信息
			wg.Done()
			detailedRunners[i] = runner
			continue
		case sem <- struct{}{}: // Acquire semaphore
		}
		
		go func(index int, r RunnerInfo) {
			defer func() {
				<-sem // Release semaphore
				wg.Done()
			}()

			// 使用带上下文的 GetRunner
			detailed, err := s.getRunnerWithContext(ctx, r.ID)
			if err != nil {
				// If we can't get detailed info, use basic info from list
				if err != context.Canceled && err != context.DeadlineExceeded {
					s.logger.Warning("Failed to get detailed info for runner " + fmt.Sprintf("%d", r.ID) + ": " + err.Error())
				}
				detailedRunners[index] = r
				return
			}
			detailedRunners[index] = *detailed
		}(i, runner)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Mark which runners are created by platform
	// Get all platform-created runner IDs from database
	var platformRunnerIDs []int
	err = s.db.Model(&model.GitlabRunner{}).Pluck("runner_id", &platformRunnerIDs).Error
	if err != nil {
		s.logger.Warning(fmt.Sprintf("Failed to get platform runner IDs: %v", err))
	}

	// Create a map for quick lookup
	platformRunnerMap := make(map[int]bool)
	for _, id := range platformRunnerIDs {
		platformRunnerMap[id] = true
	}

	// Convert detailedRunners to map[string]interface{} to add is_platform_created field
	result := make([]map[string]interface{}, len(detailedRunners))
	for i, runner := range detailedRunners {
		// Convert runner to map
		data, _ := json.Marshal(runner)
		var runnerMap map[string]interface{}
		json.Unmarshal(data, &runnerMap)

		// Add is_platform_created field
		runnerMap["is_platform_created"] = platformRunnerMap[runner.ID]
		result[i] = runnerMap
	}

	return result, nil
}

// PipelineInfo represents GitLab pipeline information
type PipelineInfo struct {
	ID             int       `json:"id"`
	ProjectID      int       `json:"project_id"`
	Status         string    `json:"status"`
	Ref            string    `json:"ref"`
	SHA            string    `json:"sha"`
	WebURL         string    `json:"web_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
	Duration       int       `json:"duration"`
	QueuedDuration int       `json:"queued_duration"` // Time spent in queue before execution
}

// ListPipelines retrieves pipelines for a project with pagination support
func (s *Service) ListPipelines(projectID int, ref, status string, page, perPage int) ([]PipelineInfo, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if !settings.Enabled {
		return nil, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, errors.New("GitLab domain or token is not configured")
	}

	// Set default pagination values
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100 // GitLab API max per_page
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// Build URL
	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/pipelines", settings.Domain, projectID)
	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	if ref != "" {
		q.Set("ref", ref)
	}
	if status != "" {
		q.Set("status", status)
	}
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	q.Set("page", fmt.Sprintf("%d", page))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", settings.Token)

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, req, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
	}

	var pipelines []PipelineInfo
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &pipelines); err != nil {
		return nil, err
	}

	// Calculate duration and queued_duration if not provided by GitLab API
	// Note: GitLab list API doesn't return started_at/finished_at, so we use calculated values
	for i := range pipelines {
		// Calculate duration if not provided
		if pipelines[i].Duration == 0 {
			// Check if we have valid timestamps
			if !pipelines[i].FinishedAt.IsZero() && !pipelines[i].StartedAt.IsZero() {
				// If we have finished_at and started_at, use them (most accurate)
				duration := pipelines[i].FinishedAt.Sub(pipelines[i].StartedAt)
				pipelines[i].Duration = int(duration.Seconds())
			} else if !pipelines[i].UpdatedAt.IsZero() && !pipelines[i].CreatedAt.IsZero() {
				// Otherwise, use updated_at - created_at (approximate)
				duration := pipelines[i].UpdatedAt.Sub(pipelines[i].CreatedAt)
				pipelines[i].Duration = int(duration.Seconds())
			}
		}

		// Calculate queued_duration if not provided
		// Queued duration is typically: started_at - created_at
		if pipelines[i].QueuedDuration == 0 && !pipelines[i].CreatedAt.IsZero() {
			// If we have started_at, calculate the queued time
			if !pipelines[i].StartedAt.IsZero() {
				queuedDuration := pipelines[i].StartedAt.Sub(pipelines[i].CreatedAt)
				if queuedDuration > 0 {
					pipelines[i].QueuedDuration = int(queuedDuration.Seconds())
				}
			}
			// Note: If started_at is not available (zero value), we cannot calculate queued_duration
			// In this case, queued_duration will remain 0 or show as "-" in the UI
		}
	}

	s.logger.Info(fmt.Sprintf("Fetched %d pipelines from GitLab for project %d (page=%d, per_page=%d)", len(pipelines), projectID, page, perPage))

	return pipelines, nil
}

// PipelineDetailInfo represents detailed GitLab pipeline information
type PipelineDetailInfo struct {
	ID             int                    `json:"id"`
	ProjectID      int                    `json:"project_id"`
	Status         string                 `json:"status"`
	Ref            string                 `json:"ref"`
	SHA            string                 `json:"sha"`
	WebURL         string                 `json:"web_url"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	StartedAt      *time.Time             `json:"started_at"`
	FinishedAt     *time.Time             `json:"finished_at"`
	Duration       int                    `json:"duration"`
	QueuedDuration int                    `json:"queued_duration"`
	Coverage       *float64               `json:"coverage"`
	User           map[string]interface{} `json:"user"`
}

// PipelineJobInfo represents a job in a pipeline
type PipelineJobInfo struct {
	ID             int                    `json:"id"`
	Status         string                 `json:"status"`
	Stage          string                 `json:"stage"`
	Name           string                 `json:"name"`
	Ref            string                 `json:"ref"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      *time.Time             `json:"started_at"`
	FinishedAt     *time.Time             `json:"finished_at"`
	Duration       float64                `json:"duration"`
	QueuedDuration float64                `json:"queued_duration"`
	WebURL         string                 `json:"web_url"`
	User           map[string]interface{} `json:"user"`
}

// GetPipelineDetail retrieves detailed information for a specific pipeline
func (s *Service) GetPipelineDetail(projectID, pipelineID int) (*PipelineDetailInfo, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if !settings.Enabled {
		return nil, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, errors.New("GitLab domain or token is not configured")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/pipelines/%d", settings.Domain, projectID, pipelineID)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", settings.Token)

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, req, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pipelineDetail PipelineDetailInfo
	if err := json.Unmarshal(body, &pipelineDetail); err != nil {
		return nil, err
	}

	// Calculate queued_duration if not provided and we have valid timestamps
	if pipelineDetail.QueuedDuration == 0 && pipelineDetail.StartedAt != nil && !pipelineDetail.CreatedAt.IsZero() {
		queuedDuration := pipelineDetail.StartedAt.Sub(pipelineDetail.CreatedAt)
		if queuedDuration > 0 {
			pipelineDetail.QueuedDuration = int(queuedDuration.Seconds())
		}
	}

	s.logger.Info(fmt.Sprintf("Fetched pipeline detail: ID=%d, Duration=%d, QueuedDuration=%d",
		pipelineDetail.ID, pipelineDetail.Duration, pipelineDetail.QueuedDuration))

	return &pipelineDetail, nil
}

// GetPipelineJobs retrieves jobs for a specific pipeline
func (s *Service) GetPipelineJobs(projectID, pipelineID int) ([]PipelineJobInfo, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if !settings.Enabled {
		return nil, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, errors.New("GitLab domain or token is not configured")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/pipelines/%d/jobs", settings.Domain, projectID, pipelineID)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", settings.Token)

	// Set per_page to get all jobs (max 100)
	q := req.URL.Query()
	q.Set("per_page", "100")
	req.URL.RawQuery = q.Encode()

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, req, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jobs []PipelineJobInfo
	if err := json.Unmarshal(body, &jobs); err != nil {
		return nil, err
	}

	s.logger.Info(fmt.Sprintf("Fetched %d jobs for pipeline %d", len(jobs), pipelineID))

	return jobs, nil
}

// RunnerJobInfo represents a job run by a specific runner
type RunnerJobInfo struct {
	ID             int                    `json:"id"`
	Status         string                 `json:"status"`
	Stage          string                 `json:"stage"`
	Name           string                 `json:"name"`
	Ref            string                 `json:"ref"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      *time.Time             `json:"started_at"`
	FinishedAt     *time.Time             `json:"finished_at"`
	Duration       float64                `json:"duration"`
	QueuedDuration float64                `json:"queued_duration"`
	WebURL         string                 `json:"web_url"`
	Pipeline       map[string]interface{} `json:"pipeline"`
	Project        map[string]interface{} `json:"project"`
	User           map[string]interface{} `json:"user"`
}

// GlobalJobInfo represents a job from all visible projects
type GlobalJobInfo struct {
	ID             int                    `json:"id"`
	Name           string                 `json:"name"`
	Status         string                 `json:"status"`
	Stage          string                 `json:"stage"`
	Ref            string                 `json:"ref"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      *time.Time             `json:"started_at"`
	FinishedAt     *time.Time             `json:"finished_at"`
	Duration       float64                `json:"duration"`
	QueuedDuration float64                `json:"queued_duration"`
	WebURL         string                 `json:"web_url"`
	TagList        []string               `json:"tag_list"`
	Pipeline       map[string]interface{} `json:"pipeline"`
	Project        map[string]interface{} `json:"project"`
	Runner         map[string]interface{} `json:"runner"` // May be nil
	User           map[string]interface{} `json:"user"`
	Commit         map[string]interface{} `json:"commit"`
}

// GetRunnerJobs retrieves jobs run by a specific runner
func (s *Service) GetRunnerJobs(runnerID int, status string, page, perPage int) ([]RunnerJobInfo, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if !settings.Enabled {
		return nil, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, errors.New("GitLab domain or token is not configured")
	}

	// Set default pagination values
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100 // GitLab API max per_page
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf("%s/api/v4/runners/%d/jobs", settings.Domain, runnerID)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", settings.Token)

	// Set query parameters
	q := req.URL.Query()
	if status != "" {
		q.Set("status", status)
	}
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	q.Set("page", fmt.Sprintf("%d", page))
	req.URL.RawQuery = q.Encode()

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, req, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jobs []RunnerJobInfo
	if err := json.Unmarshal(body, &jobs); err != nil {
		return nil, err
	}

	s.logger.Info(fmt.Sprintf("Fetched %d jobs for runner %d (page=%d, per_page=%d)", len(jobs), runnerID, page, perPage))

	return jobs, nil
}

// GetRunner retrieves a specific runner by ID
func (s *Service) GetRunner(runnerID int) (*RunnerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.getRunnerWithContext(ctx, runnerID)
}

// getRunnerWithContext retrieves a specific runner by ID with context support
func (s *Service) getRunnerWithContext(ctx context.Context, runnerID int) (*RunnerInfo, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if !settings.Enabled {
		return nil, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, errors.New("GitLab domain or token is not configured")
	}

	apiURL := fmt.Sprintf("%s/api/v4/runners/%d", settings.Domain, runnerID)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", settings.Token)

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, req, 1) // 减少重试次数
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
	}

	var runner RunnerInfo
	if err := json.NewDecoder(resp.Body).Decode(&runner); err != nil {
		return nil, err
	}

	return &runner, nil
}

// UpdateRunner updates a runner's configuration
func (s *Service) UpdateRunner(runnerID int, req UpdateRunnerRequest) (*RunnerInfo, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if !settings.Enabled {
		return nil, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, errors.New("GitLab domain or token is not configured")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/api/v4/runners/%d", settings.Domain, runnerID)
	httpReq, err := http.NewRequest("PUT", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("PRIVATE-TOKEN", settings.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, httpReq, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
	}

	var runner RunnerInfo
	if err := json.NewDecoder(resp.Body).Decode(&runner); err != nil {
		return nil, err
	}

	return &runner, nil
}

// DeleteRunner deletes a runner
func (s *Service) DeleteRunner(runnerID int) error {
	settings, err := s.GetSettings()
	if err != nil {
		return err
	}

	if !settings.Enabled {
		return errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return errors.New("GitLab domain or token is not configured")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf("%s/api/v4/runners/%d", settings.Domain, runnerID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", settings.Token)

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, req, 2)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Delete runner token from database
	if err := s.DeleteRunnerToken(runnerID); err != nil {
		s.logger.Warning(fmt.Sprintf("Failed to delete runner token from database: %v", err))
		// Don't fail the entire operation
	}

	return nil
}

// CreateRunner creates a new runner in GitLab
func (s *Service) CreateRunner(req CreateRunnerRequest, username string) (*CreateRunnerResponse, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if !settings.Enabled {
		return nil, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, errors.New("GitLab domain or token is not configured")
	}

	// Validate required fields based on runner type
	switch req.RunnerType {
	case "group_type":
		if req.GroupID == nil {
			return nil, errors.New("group_id is required for group_type runner")
		}
	case "project_type":
		if req.ProjectID == nil {
			return nil, errors.New("project_id is required for project_type runner")
		}
	case "instance_type":
		// No additional validation needed
	default:
		return nil, errors.New("invalid runner_type, must be one of: instance_type, group_type, project_type")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build request body
	data := url.Values{}
	data.Set("runner_type", req.RunnerType)
	if req.GroupID != nil {
		data.Set("group_id", fmt.Sprintf("%d", *req.GroupID))
	}
	if req.ProjectID != nil {
		data.Set("project_id", fmt.Sprintf("%d", *req.ProjectID))
	}
	if req.Description != "" {
		data.Set("description", req.Description)
	}
	if len(req.TagList) > 0 {
		data.Set("tag_list", strings.Join(req.TagList, ","))
	}
	if req.RunUntagged != nil {
		data.Set("run_untagged", fmt.Sprintf("%t", *req.RunUntagged))
	}
	if req.Locked != nil {
		data.Set("locked", fmt.Sprintf("%t", *req.Locked))
	}
	if req.AccessLevel != nil {
		data.Set("access_level", *req.AccessLevel)
	}
	if req.MaximumTimeout != nil {
		data.Set("maximum_timeout", fmt.Sprintf("%d", *req.MaximumTimeout))
	}
	if req.Paused != nil {
		data.Set("paused", fmt.Sprintf("%t", *req.Paused))
	}

	apiURL := fmt.Sprintf("%s/api/v4/user/runners", settings.Domain)
	httpReq, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("PRIVATE-TOKEN", settings.Token)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, httpReq, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
	}

	var createResp CreateRunnerResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, err
	}

	s.logger.Info(fmt.Sprintf("Created new runner: ID=%d, Type=%s, Description=%s", createResp.ID, createResp.RunnerType, createResp.Description))

	// Save runner token to database
	if err := s.SaveRunnerToken(createResp.ID, createResp.Token, createResp.Description, createResp.RunnerType, username); err != nil {
		s.logger.Warning(fmt.Sprintf("Failed to save runner token to database: %v", err))
		// Don't fail the entire operation if we can't save the token
	}

	return &createResp, nil
}

// ResetRunnerToken resets a runner's authentication token in GitLab
func (s *Service) ResetRunnerToken(runnerID int, username string) (*CreateRunnerResponse, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	if !settings.Enabled {
		return nil, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, errors.New("GitLab domain or token is not configured")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf("%s/api/v4/runners/%d/reset_authentication_token", settings.Domain, runnerID)
	httpReq, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("PRIVATE-TOKEN", settings.Token)

	// 使用带重试的请求
	resp, err := s.doHTTPRequestWithRetry(ctx, httpReq, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
	}

	var resetResp CreateRunnerResponse
	if err := json.NewDecoder(resp.Body).Decode(&resetResp); err != nil {
		return nil, err
	}

	// GitLab API may not return ID in reset response, use the parameter instead
	if resetResp.ID == 0 {
		resetResp.ID = runnerID
		s.logger.Info(fmt.Sprintf("Reset token response ID is 0, using parameter runner ID=%d", runnerID))
	}

	s.logger.Info(fmt.Sprintf("Reset token for runner: ID=%d, Token length=%d", resetResp.ID, len(resetResp.Token)))

	// Update runner token in database
	if err := s.UpdateRunnerToken(resetResp.ID, resetResp.Token); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to update runner token in database: %v", err))
		return nil, fmt.Errorf("failed to update runner token in database: %w", err)
	}

	s.logger.Info(fmt.Sprintf("Successfully updated runner token in database for runner ID=%d", resetResp.ID))

	return &resetResp, nil
}

// ProjectBasicInfo represents basic project information
type ProjectBasicInfo struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	NameWithNamespace string `json:"name_with_namespace"`
	PathWithNamespace string `json:"path_with_namespace"`
}

// projectJobsResult 表示项目 jobs 获取的结果
type projectJobsResult struct {
	ProjectID   int
	ProjectName string
	Jobs        []GlobalJobInfo
	Error       error
}

// ListAllJobs retrieves all visible jobs across all projects
// Returns: jobs, totalCount, filteredCount, error
func (s *Service) ListAllJobs(status, tag string, page, perPage int) ([]GlobalJobInfo, int, int, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, 0, 0, err
	}

	if !settings.Enabled {
		return nil, 0, 0, errors.New("GitLab integration is not enabled")
	}

	if settings.Domain == "" || settings.Token == "" {
		return nil, 0, 0, errors.New("GitLab domain or token is not configured")
	}

	// Set default pagination values
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100 // GitLab API max per_page
	}

	// 创建带超时的上下文，总体操作不超过 60 秒
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 获取所有用户的项目（支持分页）
	projects, err := s.fetchAllUserProjects(ctx, settings)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to fetch projects: %w", err)
	}

	if len(projects) == 0 {
		s.logger.Info("No projects found for user")
		return []GlobalJobInfo{}, 0, 0, nil
	}

	s.logger.Info(fmt.Sprintf("Found %d projects, fetching jobs...", len(projects)))

	startTime := time.Now()

	// 根据状态过滤条件确定搜索策略
	searchStrategy := s.determineSearchStrategy(status)
	
	// 检查是否是不支持的状态
	if searchStrategy.MaxProjects == 0 {
		s.logger.Warning(fmt.Sprintf("Unsupported status filter: %s", status))
		return []GlobalJobInfo{}, 0, 0, fmt.Errorf("状态 '%s' 不支持直接查询。建议：1) 不使用状态过滤查看活跃的 jobs，或 2) 使用前端表格的状态筛选功能", status)
	}
	
	s.logger.Info(fmt.Sprintf("Search strategy: time_range=%s, scopes=%v, max_pages=%d",
		searchStrategy.TimeRange, searchStrategy.Scopes, searchStrategy.MaxPagesPerProject))

	// 并发控制参数
	maxProjectsLimit := searchStrategy.MaxProjects // 根据策略动态调整
	maxConcurrency := 15                           // 增加并发数以提高速度
	maxPagesPerProject := searchStrategy.MaxPagesPerProject
	
	// 限制要处理的项目数量（如果需要）
	originalProjectCount := len(projects)
	if maxProjectsLimit > 0 && len(projects) > maxProjectsLimit {
		projects = projects[:maxProjectsLimit]
		s.logger.Info(fmt.Sprintf("Limiting projects from %d to %d (maxProjectsLimit)", originalProjectCount, maxProjectsLimit))
	}

	// 使用通道和 WaitGroup 进行并发控制
	resultChan := make(chan projectJobsResult, len(projects))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)
	
	// 选择并发模式：
	// 1. 标准模式：项目级并发 + 项目内分页并发（默认）
	// 2. 高级模式：全局 Worker Pool，所有请求打平处理（可选，更快但更耗资源）
	useAdvancedMode := len(projects) > 30 && maxPagesPerProject >= 3
	
	if useAdvancedMode {
		s.logger.Info(fmt.Sprintf("Using advanced worker pool mode for %d projects", len(projects)))
		// 使用全局 Worker Pool 模式
		s.fetchJobsWithWorkerPool(ctx, settings, projects, searchStrategy, maxPagesPerProject, resultChan, &wg)
	} else {
		// 并发获取每个项目的 jobs（标准模式）
		for _, project := range projects {
			wg.Add(1)
			go func(proj ProjectBasicInfo) {
				defer wg.Done()
				
				// 获取信号量
				select {
				case semaphore <- struct{}{}:
					defer func() { <-semaphore }()
				case <-ctx.Done():
					resultChan <- projectJobsResult{
						ProjectID: proj.ID,
						Error:     ctx.Err(),
					}
					return
				}
				
				// 为该项目获取 jobs（使用项目内分页并发）
				projectJobs := s.fetchProjectJobs(ctx, settings, proj, searchStrategy, maxPagesPerProject)
				
				// 发送结果
				resultChan <- projectJobsResult{
					ProjectID:   proj.ID,
					ProjectName: proj.Name,
					Jobs:        projectJobs,
					Error:       nil,
				}
			}(project)
		}
	}
	
	// 等待所有 goroutine 完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// 收集结果
	var allJobs []GlobalJobInfo
	projectsProcessed := 0
	projectsFailed := 0
	totalJobsCollected := 0
	
	for result := range resultChan {
		if result.Error != nil {
			projectsFailed++
			if result.Error != context.Canceled && result.Error != context.DeadlineExceeded {
				s.logger.Warning(fmt.Sprintf("Failed to fetch jobs for project %d: %v", result.ProjectID, result.Error))
			}
			continue
		}
		
		if len(result.Jobs) > 0 {
			projectsProcessed++
			totalJobsCollected += len(result.Jobs)
			allJobs = append(allJobs, result.Jobs...)
		}
	}

	elapsedTime := time.Since(startTime)
	s.logger.Info(fmt.Sprintf("Processed %d projects (%d failed), collected %d jobs in %.2f seconds", 
		projectsProcessed, projectsFailed, totalJobsCollected, elapsedTime.Seconds()))

	// Collect status statistics for debugging
	statusCounts := make(map[string]int)
	for _, job := range allJobs {
		statusCounts[job.Status]++
	}
	s.logger.Info(fmt.Sprintf("[ListAllJobs] Status distribution (before filtering): %v", statusCounts))

	// Record total count before filtering
	totalCount := len(allJobs)
	if totalCount > 10000 {
		totalCount = 10001 // Signal that there are more than 10000
	}

	// Filter by status if specified (in memory)
	if status != "" {
		var statusFilteredJobs []GlobalJobInfo
		statusLower := strings.ToLower(status)

		s.logger.Info(fmt.Sprintf("[ListAllJobs] Filtering by status: %s", statusLower))
		
		for _, job := range allJobs {
			if strings.ToLower(job.Status) == statusLower {
				statusFilteredJobs = append(statusFilteredJobs, job)
			}
		}

		s.logger.Info(fmt.Sprintf("[ListAllJobs] After status filtering: found %d jobs with status '%s' (from %d total)", 
			len(statusFilteredJobs), statusLower, len(allJobs)))
		
		allJobs = statusFilteredJobs
	}

	// Filter by tag if specified (in memory)
	if tag != "" {
		var tagFilteredJobs []GlobalJobInfo
		tagLower := strings.ToLower(tag)

		for _, job := range allJobs {
			// Check if any tag in job's tag_list contains the search tag
			if len(job.TagList) > 0 {
				for _, jobTag := range job.TagList {
					if strings.Contains(strings.ToLower(jobTag), tagLower) {
						tagFilteredJobs = append(tagFilteredJobs, job)
						break
					}
				}
			}
		}

		allJobs = tagFilteredJobs
	}

	// Record filtered count after all filters applied
	filteredCount := len(allJobs)

	// Sort jobs by created_at (newest first) using efficient sort
	sort.Slice(allJobs, func(i, j int) bool {
		return allJobs[i].CreatedAt.After(allJobs[j].CreatedAt)
	})

	// Apply pagination to collected jobs
	startIdx := (page - 1) * perPage
	endIdx := startIdx + perPage

	if startIdx >= len(allJobs) {
		return []GlobalJobInfo{}, totalCount, filteredCount, nil
	}

	if endIdx > len(allJobs) {
		endIdx = len(allJobs)
	}

	result := allJobs[startIdx:endIdx]

	// Log summary
	filters := make([]string, 0)
	if status != "" {
		filters = append(filters, fmt.Sprintf("status=%s", status))
	}
	if tag != "" {
		filters = append(filters, fmt.Sprintf("tag=%s", tag))
	}
	filterStr := ""
	if len(filters) > 0 {
		filterStr = fmt.Sprintf(" [%s]", strings.Join(filters, ", "))
	}

	s.logger.Info(fmt.Sprintf("[ListAllJobs] Total: %d, Filtered: %d, Page: %d, Returning: %d jobs%s",
		totalCount, filteredCount, page, len(result), filterStr))

	return result, totalCount, filteredCount, nil
}

// SearchStrategy 定义搜索策略
type SearchStrategy struct {
	TimeRange          string   // 时间范围描述
	TimeRangeDays      int      // 时间范围（天）
	Scopes             []string // GitLab API scope 参数
	MaxPagesPerProject int      // 每个项目最多获取的页数
	MaxProjects        int      // 最多处理的项目数（0 表示不限制）
}

// determineSearchStrategy 根据状态过滤条件确定搜索策略
func (s *Service) determineSearchStrategy(status string) SearchStrategy {
	statusLower := strings.ToLower(status)
	
	// 根据不同的状态选择不同的搜索策略
	switch statusLower {
	case "created", "pending", "running":
		// 活跃状态（核心状态）：最近3天
		return SearchStrategy{
			TimeRange:          "last 3 days",
			TimeRangeDays:      3,
			Scopes:             []string{"created", "pending", "running"},
			MaxPagesPerProject: 3,
			MaxProjects:        80,
		}
	
	case "preparing", "scheduled":
		// 活跃状态（准备中）：最近3天
		return SearchStrategy{
			TimeRange:          "last 3 days",
			TimeRangeDays:      3,
			Scopes:             []string{statusLower},
			MaxPagesPerProject: 2,
			MaxProjects:        80,
		}
	
	case "waiting_for_resource":
		// 等待资源状态：只获取活跃状态，避免获取已完成的 jobs
		// GitLab API 不支持 waiting_for_resource 作为 scope，所以获取活跃状态后在内存中过滤
		return SearchStrategy{
			TimeRange:          "last 3 days",
			TimeRangeDays:      3,
			Scopes:             []string{"created", "pending", "running"}, // 只获取活跃状态
			MaxPagesPerProject: 3,
			MaxProjects:        50,
		}
	
	case "success", "failed", "canceled", "skipped":
		// 终止状态：不再支持直接查询，提示用户使用无过滤查询
		// 返回空策略，在调用处检测并返回友好提示
		s.logger.Warning(fmt.Sprintf("Status '%s' is not supported for direct query. Please use no filter to get all jobs.", statusLower))
		return SearchStrategy{
			TimeRange:          "not supported",
			TimeRangeDays:      0,
			Scopes:             []string{},
			MaxPagesPerProject: 0,
			MaxProjects:        0,
		}
	
	case "manual":
		// 手动状态：最近7天（减少时间范围）
		return SearchStrategy{
			TimeRange:          "last 7 days",
			TimeRangeDays:      7,
			Scopes:             []string{"manual"},
			MaxPagesPerProject: 2,
			MaxProjects:        60,
		}
	
	case "":
		// 无状态过滤：只获取活跃状态
		return SearchStrategy{
			TimeRange:          "last 3 days",
			TimeRangeDays:      3,
			Scopes:             []string{"created", "pending", "running"},
			MaxPagesPerProject: 3,
			MaxProjects:        80,
		}
	
	default:
		// 其他状态：尝试使用该状态作为 scope
		return SearchStrategy{
			TimeRange:          "last 3 days",
			TimeRangeDays:      3,
			Scopes:             []string{statusLower},
			MaxPagesPerProject: 3,
			MaxProjects:        70,
		}
	}
}

// fetchAllUserProjects 获取用户的所有项目（支持分页）
func (s *Service) fetchAllUserProjects(ctx context.Context, settings *model.GitlabSettings) ([]ProjectBasicInfo, error) {
	var allProjects []ProjectBasicInfo
	page := 1
	perPage := 100 // GitLab API 最大值
	
	for {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		projectsURL := fmt.Sprintf("%s/api/v4/projects", settings.Domain)
		req, err := http.NewRequest("GET", projectsURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("PRIVATE-TOKEN", settings.Token)

		q := req.URL.Query()
		q.Set("membership", "true")
		q.Set("per_page", fmt.Sprintf("%d", perPage))
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("simple", "true")
		q.Set("archived", "false")
		q.Set("order_by", "last_activity_at")
		q.Set("sort", "desc")
		req.URL.RawQuery = q.Encode()

		resp, err := s.doHTTPRequestWithRetry(ctx, req, 2)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var projects []ProjectBasicInfo
		if err := json.Unmarshal(body, &projects); err != nil {
			return nil, fmt.Errorf("failed to parse projects: %w", err)
		}

		allProjects = append(allProjects, projects...)
		
		// 如果返回的项目数少于 perPage，说明已经是最后一页
		if len(projects) < perPage {
			break
		}
		
		// 限制最多获取 300 个项目（3 页）以避免超时
		if len(allProjects) >= 300 {
			s.logger.Info(fmt.Sprintf("Reached project limit (300), stopping pagination"))
			break
		}
		
		page++
	}
	
	s.logger.Info(fmt.Sprintf("Fetched %d projects in total", len(allProjects)))
	return allProjects, nil
}

// JobFetchTask 表示一个 jobs 获取任务
type JobFetchTask struct {
	ProjectID   int
	ProjectName string
	ProjectInfo ProjectBasicInfo
	PageNum     int
	Strategy    SearchStrategy
}

// JobFetchResult 表示 jobs 获取任务的结果
type JobFetchResult struct {
	ProjectID  int
	PageNum    int
	Jobs       []GlobalJobInfo
	HasMore    bool // 是否还有更多页
	Error      error
}

// fetchProjectJobsConcurrent 使用并发方式获取单个项目的 jobs（项目内分页并发）
func (s *Service) fetchProjectJobsConcurrent(ctx context.Context, settings *model.GitlabSettings, 
	proj ProjectBasicInfo, strategy SearchStrategy, maxPages int) []GlobalJobInfo {
	
	if maxPages <= 1 {
		// 如果只有一页，不需要并发
		return s.fetchProjectJobsSinglePage(ctx, settings, proj, strategy, 1)
	}

	timeThreshold := time.Now().AddDate(0, 0, -strategy.TimeRangeDays)
	
	// 使用 channel 进行并发控制
	type pageResult struct {
		pageNum int
		jobs    []GlobalJobInfo
		hasMore bool
		err     error
	}
	
	resultChan := make(chan pageResult, maxPages)
	var wg sync.WaitGroup
	
	// 并发获取前几页（通常前2-3页就包含了大部分活跃的 jobs）
	concurrentPages := maxPages
	if concurrentPages > 3 {
		concurrentPages = 3 // 前3页并发获取
	}
	
	// 启动并发任务获取前几页
	for pageNum := 1; pageNum <= concurrentPages; pageNum++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			
			// 检查上下文
			select {
			case <-ctx.Done():
				resultChan <- pageResult{pageNum: page, err: ctx.Err()}
				return
			default:
			}
			
			jobs, hasMore, err := s.fetchSingleJobPage(ctx, settings, proj, strategy, page, timeThreshold)
			resultChan <- pageResult{
				pageNum: page,
				jobs:    jobs,
				hasMore: hasMore,
				err:     err,
			}
		}(pageNum)
	}
	
	// 等待并发任务完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// 收集结果
	pageResults := make(map[int]pageResult)
	for result := range resultChan {
		pageResults[result.pageNum] = result
	}
	
	// 按页码顺序组装结果
	var allJobs []GlobalJobInfo
	shouldContinue := true
	
	for pageNum := 1; pageNum <= concurrentPages && shouldContinue; pageNum++ {
		result, exists := pageResults[pageNum]
		if !exists || result.err != nil {
			if pageNum == 1 {
				s.logger.Debug(fmt.Sprintf("Failed to fetch jobs for project %d page %d", proj.ID, pageNum))
			}
			break
		}
		
		allJobs = append(allJobs, result.jobs...)
		
		// 如果这一页没有更多数据，停止
		if !result.hasMore {
			shouldContinue = false
			break
		}
	}
	
	// 如果还需要获取更多页，串行获取剩余页（通常不会到这里）
	if shouldContinue && len(pageResults) > 0 {
		lastPage := pageResults[concurrentPages]
		if lastPage.hasMore {
			for pageNum := concurrentPages + 1; pageNum <= maxPages; pageNum++ {
				jobs, hasMore, err := s.fetchSingleJobPage(ctx, settings, proj, strategy, pageNum, timeThreshold)
				if err != nil || len(jobs) == 0 {
					break
				}
				allJobs = append(allJobs, jobs...)
				if !hasMore {
					break
				}
			}
		}
	}
	
	return allJobs
}

// fetchSingleJobPage 获取单页 jobs 数据
func (s *Service) fetchSingleJobPage(ctx context.Context, settings *model.GitlabSettings,
	proj ProjectBasicInfo, strategy SearchStrategy, pageNum int, timeThreshold time.Time) ([]GlobalJobInfo, bool, error) {
	
	// 检查上下文
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}
	
	// 构建请求
	jobsURL := fmt.Sprintf("%s/api/v4/projects/%d/jobs", settings.Domain, proj.ID)
	jobReq, err := http.NewRequest("GET", jobsURL, nil)
	if err != nil {
		return nil, false, err
	}
	jobReq.Header.Set("PRIVATE-TOKEN", settings.Token)

	jobQ := jobReq.URL.Query()
	
	// 根据策略设置 scope 过滤
	if len(strategy.Scopes) > 0 {
		for _, scope := range strategy.Scopes {
			jobQ.Add("scope[]", scope)
		}
	}
	
	jobQ.Set("per_page", "100")
	jobQ.Set("page", fmt.Sprintf("%d", pageNum))
	jobQ.Set("order_by", "id")
	jobQ.Set("sort", "desc")
	jobReq.URL.RawQuery = jobQ.Encode()

	// 使用带重试的请求
	jobResp, err := s.doHTTPRequestWithRetry(ctx, jobReq, 1)
	if err != nil {
		return nil, false, err
	}
	defer jobResp.Body.Close()

	if jobResp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("status code: %d", jobResp.StatusCode)
	}

	jobBody, err := io.ReadAll(jobResp.Body)
	if err != nil {
		return nil, false, err
	}

	var pageJobs []GlobalJobInfo
	if err := json.Unmarshal(jobBody, &pageJobs); err != nil {
		return nil, false, fmt.Errorf("parse error: %w", err)
	}

	// 如果没有返回作业，说明到达最后一页
	if len(pageJobs) == 0 {
		return nil, false, nil
	}

	// 过滤时间范围并丰富项目信息
	var filteredJobs []GlobalJobInfo
	hasOldJobs := false
	
	for i := range pageJobs {
		// 检查作业是否在时间范围内
		if pageJobs[i].CreatedAt.Before(timeThreshold) {
			hasOldJobs = true
			continue
		}

		// 丰富项目信息
		if len(pageJobs[i].Project) == 0 {
			pageJobs[i].Project = map[string]interface{}{
				"id":                  proj.ID,
				"name":                proj.Name,
				"name_with_namespace": proj.NameWithNamespace,
				"path_with_namespace": proj.PathWithNamespace,
			}
		}

		filteredJobs = append(filteredJobs, pageJobs[i])
	}

	// 判断是否还有更多页
	// 如果发现旧作业或返回的作业少于 100 个，说明没有更多页了
	hasMore := !hasOldJobs && len(pageJobs) >= 100
	
	return filteredJobs, hasMore, nil
}

// fetchProjectJobsSinglePage 获取单页数据（简化版，用于只需要1页的情况）
func (s *Service) fetchProjectJobsSinglePage(ctx context.Context, settings *model.GitlabSettings,
	proj ProjectBasicInfo, strategy SearchStrategy, pageNum int) []GlobalJobInfo {
	
	timeThreshold := time.Now().AddDate(0, 0, -strategy.TimeRangeDays)
	jobs, _, _ := s.fetchSingleJobPage(ctx, settings, proj, strategy, pageNum, timeThreshold)
	return jobs
}

// fetchProjectJobs 获取单个项目的 jobs（保留原函数名以兼容，内部使用并发版本）
func (s *Service) fetchProjectJobs(ctx context.Context, settings *model.GitlabSettings, 
	proj ProjectBasicInfo, strategy SearchStrategy, maxPages int) []GlobalJobInfo {
	
	// 使用并发版本
	return s.fetchProjectJobsConcurrent(ctx, settings, proj, strategy, maxPages)
}

// fetchJobsWithWorkerPool 使用 Worker Pool 模式并发获取所有 jobs（全局任务调度）
func (s *Service) fetchJobsWithWorkerPool(ctx context.Context, settings *model.GitlabSettings,
	projects []ProjectBasicInfo, strategy SearchStrategy, maxPages int,
	resultChan chan<- projectJobsResult, wg *sync.WaitGroup) {
	
	timeThreshold := time.Now().AddDate(0, 0, -strategy.TimeRangeDays)
	
	// 创建任务队列
	type jobTask struct {
		project ProjectBasicInfo
		page    int
	}
	
	taskChan := make(chan jobTask, len(projects)*maxPages)
	
	// Worker Pool：30个并发 worker（增加并发度）
	workerCount := 30
	if workerCount > len(projects)*maxPages {
		workerCount = len(projects) * maxPages
	}
	
	// 页面结果收集
	type pageTaskResult struct {
		projectID int
		page      int
		jobs      []GlobalJobInfo
		hasMore   bool
		err       error
	}
	
	pageResultChan := make(chan pageTaskResult, len(projects)*maxPages)
	
	// 启动 workers
	var workerWg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		workerWg.Add(1)
		go func(workerID int) {
			defer workerWg.Done()
			
			for task := range taskChan {
				// 检查上下文
				select {
				case <-ctx.Done():
					pageResultChan <- pageTaskResult{
						projectID: task.project.ID,
						page:      task.page,
						err:       ctx.Err(),
					}
					continue
				default:
				}
				
				// 获取这一页的数据
				jobs, hasMore, err := s.fetchSingleJobPage(ctx, settings, task.project, strategy, task.page, timeThreshold)
				
				pageResultChan <- pageTaskResult{
					projectID: task.project.ID,
					page:      task.page,
					jobs:      jobs,
					hasMore:   hasMore,
					err:       err,
				}
			}
		}(i)
	}
	
	// 生产者：分发初始任务（每个项目的第一页）
	go func() {
		for _, proj := range projects {
			taskChan <- jobTask{project: proj, page: 1}
		}
	}()
	
	// 预先为所有项目添加 WaitGroup 计数
	wg.Add(len(projects))
	
	// 结果收集器和动态任务分发
	go func() {
		// 记录每个项目的状态
		projectStatus := make(map[int]struct {
			currentPage int
			hasMore     bool
			jobs        []GlobalJobInfo
		})
		
		// 创建项目 ID 到项目信息的映射，提高查找效率
		projectMap := make(map[int]ProjectBasicInfo)
		for _, proj := range projects {
			projectStatus[proj.ID] = struct {
				currentPage int
				hasMore     bool
				jobs        []GlobalJobInfo
			}{currentPage: 0, hasMore: true, jobs: []GlobalJobInfo{}}
			projectMap[proj.ID] = proj
		}
		
		tasksInFlight := len(projects) // 初始任务数
		
		for tasksInFlight > 0 {
			result := <-pageResultChan
			tasksInFlight--
			
			status := projectStatus[result.projectID]
			
			if result.err == nil && len(result.jobs) > 0 {
				// 合并这一页的结果
				status.jobs = append(status.jobs, result.jobs...)
				status.currentPage = result.page
				status.hasMore = result.hasMore
				projectStatus[result.projectID] = status
				
				// 如果还有更多页且未达到最大页数，分发下一页任务
				if result.hasMore && result.page < maxPages {
					// 从 map 中获取项目信息
					proj := projectMap[result.projectID]
					taskChan <- jobTask{project: proj, page: result.page + 1}
					tasksInFlight++
				}
			} else {
				// 出错或无数据，标记为完成
				status.hasMore = false
				projectStatus[result.projectID] = status
			}
		}
		
		// 关闭任务通道，等待所有 workers 完成
		close(taskChan)
		workerWg.Wait()
		close(pageResultChan)
		
		// 汇总每个项目的结果并发送
		for projectID, status := range projectStatus {
			// 从 map 中获取项目名称
			proj := projectMap[projectID]
			
			resultChan <- projectJobsResult{
				ProjectID:   projectID,
				ProjectName: proj.Name,
				Jobs:        status.jobs,
				Error:       nil,
			}
			wg.Done() // 每发送一个项目结果，计数减一
		}
	}()
}
