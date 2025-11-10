package gitlab

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Service handles GitLab-related operations
type Service struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewService creates a new GitLab service
func NewService(db *gorm.DB, logger *logger.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
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

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
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

	// Fetch all runners with pagination
	var allRunners []RunnerInfo
	page := 1
	perPage := 100 // GitLab default max per page

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for {
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

		resp, err := client.Do(req)
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
	// Increase concurrency to 50 for better performance
	maxConcurrent := 50
	sem := make(chan struct{}, maxConcurrent)

	// Use sync.WaitGroup for better synchronization
	var wg sync.WaitGroup
	wg.Add(len(allRunners))

	for i, runner := range allRunners {
		sem <- struct{}{} // Acquire semaphore
		go func(index int, r RunnerInfo) {
			defer func() {
				<-sem // Release semaphore
				wg.Done()
			}()

			detailed, err := s.GetRunner(r.ID)
			if err != nil {
				// If we can't get detailed info, use basic info from list
				s.logger.Warning("Failed to get detailed info for runner " + fmt.Sprintf("%d", r.ID) + ": " + err.Error())
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

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
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

	apiURL := fmt.Sprintf("%s/api/v4/projects/%d/pipelines/%d", settings.Domain, projectID, pipelineID)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", settings.Token)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
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

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
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

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
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

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
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

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
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

	apiURL := fmt.Sprintf("%s/api/v4/runners/%d", settings.Domain, runnerID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", settings.Token)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
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

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
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

	apiURL := fmt.Sprintf("%s/api/v4/runners/%d/reset_authentication_token", settings.Domain, runnerID)
	httpReq, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("PRIVATE-TOKEN", settings.Token)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
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

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// First, get user's projects
	// Use /api/v4/projects?membership=true to get projects user is a member of
	projectsURL := fmt.Sprintf("%s/api/v4/projects", settings.Domain)
	req, err := http.NewRequest("GET", projectsURL, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	req.Header.Set("PRIVATE-TOKEN", settings.Token)

	q := req.URL.Query()
	q.Set("membership", "true")
	q.Set("per_page", "100")   // Get up to 100 projects (GitLab API max per page)
	q.Set("simple", "true")    // Get simplified project info
	q.Set("archived", "false") // Exclude archived projects
	q.Set("order_by", "last_activity_at")
	q.Set("sort", "desc")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to fetch projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, 0, fmt.Errorf("GitLab API returned status %d when fetching projects: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, 0, err
	}

	var projects []ProjectBasicInfo
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to parse projects: %w", err)
	}

	if len(projects) == 0 {
		s.logger.Info("No projects found for user")
		return []GlobalJobInfo{}, 0, 0, nil
	}

	s.logger.Info(fmt.Sprintf("Found %d projects, fetching jobs...", len(projects)))

	// Collect jobs from all projects
	var allJobs []GlobalJobInfo
	jobCount := 0
	maxJobsToFetch := 9999 // Increase limit to collect more jobs (will show as 1000+ if over 9999)

	// Iterate through projects and fetch jobs (with pagination)
	for _, project := range projects {
		if jobCount >= maxJobsToFetch {
			break // Limit total jobs fetched to avoid performance issues
		}

		// Fetch multiple pages of jobs from each project
		jobsPerProject := 0
		maxPagesPerProject := 3 // Fetch up to 3 pages per project (3 * 100 = 300 jobs per project)

		for pageNum := 1; pageNum <= maxPagesPerProject; pageNum++ {
			if jobCount >= maxJobsToFetch {
				break
			}

			// Fetch jobs for this project (with pagination)
			jobsURL := fmt.Sprintf("%s/api/v4/projects/%d/jobs", settings.Domain, project.ID)
			jobReq, err := http.NewRequest("GET", jobsURL, nil)
			if err != nil {
				s.logger.Warning(fmt.Sprintf("Failed to create request for project %d: %v", project.ID, err))
				break
			}
			jobReq.Header.Set("PRIVATE-TOKEN", settings.Token)

			jobQ := jobReq.URL.Query()
			// Note: Do NOT filter by status here - we need to get all jobs to calculate totalCount
			// Status filtering will be done in memory after collecting all jobs
			jobQ.Set("per_page", "100")                  // Get up to 100 jobs per page (GitLab API max)
			jobQ.Set("page", fmt.Sprintf("%d", pageNum)) // Page number
			jobQ.Set("order_by", "id")                   // Order by ID
			jobQ.Set("sort", "desc")                     // Newest first
			jobReq.URL.RawQuery = jobQ.Encode()

			jobResp, err := client.Do(jobReq)
			if err != nil {
				s.logger.Warning(fmt.Sprintf("Failed to fetch jobs for project %d page %d: %v", project.ID, pageNum, err))
				break
			}

			if jobResp.StatusCode != http.StatusOK {
				jobResp.Body.Close()
				break
			}

			jobBody, err := io.ReadAll(jobResp.Body)
			jobResp.Body.Close()
			if err != nil {
				break
			}

			var projectJobs []GlobalJobInfo
			if err := json.Unmarshal(jobBody, &projectJobs); err != nil {
				s.logger.Warning(fmt.Sprintf("Failed to parse jobs for project %d page %d: %v", project.ID, pageNum, err))
				break
			}

			// If no jobs returned, we've reached the end
			if len(projectJobs) == 0 {
				break
			}

			// Enrich jobs with project information if not present
			for i := range projectJobs {
				if projectJobs[i].Project == nil || len(projectJobs[i].Project) == 0 {
					projectJobs[i].Project = map[string]interface{}{
						"id":                  project.ID,
						"name":                project.Name,
						"name_with_namespace": project.NameWithNamespace,
						"path_with_namespace": project.PathWithNamespace,
					}
				}
			}

			allJobs = append(allJobs, projectJobs...)
			jobCount += len(projectJobs)
			jobsPerProject += len(projectJobs)

			// If we got less than 100 jobs, it's the last page for this project
			if len(projectJobs) < 100 {
				break
			}
		}

		if jobsPerProject > 0 {
			s.logger.Debug(fmt.Sprintf("Collected %d jobs from project %s (ID: %d)", jobsPerProject, project.Name, project.ID))
		}
	}

	// Record total count before filtering (cap at 1000 for display)
	totalCount := len(allJobs)

	// Collect status statistics for debugging
	statusCounts := make(map[string]int)
	for _, job := range allJobs {
		statusCounts[job.Status]++
	}

	// Log status distribution (always log to help debugging)
	s.logger.Info(fmt.Sprintf("[ListAllJobs] Collected %d jobs. Status distribution: %v", len(allJobs), statusCounts))

	if totalCount > 1000 {
		totalCount = 1001 // Signal that there are more than 1000
	}

	// Filter by status if specified (in memory)
	if status != "" {
		var statusFilteredJobs []GlobalJobInfo
		statusLower := strings.ToLower(status)

		for _, job := range allJobs {
			if strings.ToLower(job.Status) == statusLower {
				statusFilteredJobs = append(statusFilteredJobs, job)
			}
		}

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

	// Sort jobs by created_at (newest first)
	// Simple bubble sort for demonstration (in production, use sort.Slice)
	for i := 0; i < len(allJobs)-1; i++ {
		for j := i + 1; j < len(allJobs); j++ {
			if allJobs[i].CreatedAt.Before(allJobs[j].CreatedAt) {
				allJobs[i], allJobs[j] = allJobs[j], allJobs[i]
			}
		}
	}

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
