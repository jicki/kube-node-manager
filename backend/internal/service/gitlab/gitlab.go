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

// RunnerInfo represents GitLab runner information
type RunnerInfo struct {
	ID           int        `json:"id"`
	Description  string     `json:"description"`
	Active       bool       `json:"active"`
	Paused       bool       `json:"paused"`
	IsShared     bool       `json:"is_shared"`
	IPAddress    string     `json:"ip_address"`
	RunnerType   string     `json:"runner_type"`
	Name         string     `json:"name"`
	Online       bool       `json:"online"`
	Status       string     `json:"status"`
	ContactedAt  *time.Time `json:"contacted_at"`
	CreatedAt    *time.Time `json:"created_at"`
	TagList      []string   `json:"tag_list"`
	Version      string     `json:"version"`
	Architecture string     `json:"architecture"`
	Platform     string     `json:"platform"`
	Locked       bool       `json:"locked"`
	AccessLevel  string     `json:"access_level"`
}

// UpdateRunnerRequest represents the request to update a runner
type UpdateRunnerRequest struct {
	Description *string   `json:"description,omitempty"`
	Active      *bool     `json:"active,omitempty"`
	TagList     *[]string `json:"tag_list,omitempty"`
	Locked      *bool     `json:"locked,omitempty"`
	AccessLevel *string   `json:"access_level,omitempty"`
}

// ListRunners retrieves all runners from GitLab
func (s *Service) ListRunners(runnerType string, status string, paused *bool) ([]RunnerInfo, error) {
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
	q.Set("per_page", "100")
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

	// Read body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var runners []RunnerInfo
	if err := json.Unmarshal(body, &runners); err != nil {
		return nil, err
	}

	// The /runners/all endpoint returns limited information (no tag_list, contacted_at, etc.)
	// Fetch detailed info for each runner to get complete data
	// This adds overhead but provides all necessary information for the UI
	detailedRunners := make([]RunnerInfo, 0, len(runners))
	for _, runner := range runners {
		detailed, err := s.GetRunner(runner.ID)
		if err != nil {
			// If we can't get detailed info, use basic info from list
			s.logger.Warning("Failed to get detailed info for runner " + fmt.Sprintf("%d", runner.ID) + ": " + err.Error())
			detailedRunners = append(detailedRunners, runner)
			continue
		}
		detailedRunners = append(detailedRunners, *detailed)
	}

	return detailedRunners, nil
}

// PipelineInfo represents GitLab pipeline information
type PipelineInfo struct {
	ID         int       `json:"id"`
	ProjectID  int       `json:"project_id"`
	Status     string    `json:"status"`
	Ref        string    `json:"ref"`
	SHA        string    `json:"sha"`
	WebURL     string    `json:"web_url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Duration   int       `json:"duration"`
}

// ListPipelines retrieves pipelines for a project
func (s *Service) ListPipelines(projectID int, ref, status string) ([]PipelineInfo, error) {
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
	q.Set("per_page", "100")
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
	if err := json.NewDecoder(resp.Body).Decode(&pipelines); err != nil {
		return nil, err
	}

	return pipelines, nil
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

	return nil
}
