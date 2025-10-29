// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/automation"
	"kube-node-manager/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 集成测试说明：
// 运行方式: go test -tags=integration ./tests/integration/...
// 这些测试需要实际的数据库和可能的外部依赖

func setupIntegrationTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to test database")

	// 迁移所有自动化相关表
	err = db.AutoMigrate(
		&model.SSHCredential{},
		&model.AnsiblePlaybook{},
		&model.AnsibleExecution{},
		&model.Script{},
		&model.ScriptExecution{},
		&model.Workflow{},
		&model.WorkflowExecution{},
	)
	require.NoError(t, err, "Failed to migrate test database")

	return db
}

func TestCredentialManager_EndToEnd(t *testing.T) {
	db := setupIntegrationTestDB(t)
	log := logger.NewLogger("test", "debug")
	cm := automation.NewCredentialManager(db, log, "test-key-32-bytes-long-exactly")

	t.Run("Create and retrieve password credential", func(t *testing.T) {
		credential := &model.SSHCredential{
			Name:        "integration-test-password",
			Description: "Integration test credential",
			Username:    "testuser",
			AuthType:    "password",
			Password:    "secure-password-123",
			Port:        22,
		}

		// 存储
		err := cm.StoreCredential(credential)
		require.NoError(t, err)
		assert.Greater(t, credential.ID, uint(0))

		// 检索
		retrieved, err := cm.GetCredential(credential.ID)
		require.NoError(t, err)
		assert.Equal(t, credential.Username, retrieved.Username)
		assert.Equal(t, "secure-password-123", retrieved.Password)

		// 列表
		credentials, err := cm.ListCredentials()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(credentials), 1)

		// 更新
		retrieved.Description = "Updated description"
		err = cm.UpdateCredential(retrieved)
		require.NoError(t, err)

		// 验证更新
		updated, err := cm.GetCredential(credential.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", updated.Description)

		// 删除
		err = cm.DeleteCredential(credential.ID)
		require.NoError(t, err)

		// 验证删除
		_, err = cm.GetCredential(credential.ID)
		assert.Error(t, err)
	})
}

func TestAnsibleService_PlaybookLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupIntegrationTestDB(t)
	log := logger.NewLogger("test", "debug")

	// 创建 Ansible 服务所需的依赖
	credentialMgr := automation.NewCredentialManager(db, log, "test-key-32-bytes-long-exactly")

	t.Run("Create and manage playbooks", func(t *testing.T) {
		playbook := &model.AnsiblePlaybook{
			Name:        "Test Playbook",
			Description: "Integration test playbook",
			Content: `---
- name: Test Playbook
  hosts: all
  tasks:
    - name: Ping
      ping:
`,
			Category:  "test",
			IsBuiltin: false,
			IsActive:  true,
			Version:   1,
		}

		// 创建
		err := db.Create(playbook).Error
		require.NoError(t, err)
		assert.Greater(t, playbook.ID, uint(0))

		// 查询
		var retrieved model.AnsiblePlaybook
		err = db.First(&retrieved, playbook.ID).Error
		require.NoError(t, err)
		assert.Equal(t, playbook.Name, retrieved.Name)

		// 更新
		playbook.Description = "Updated test playbook"
		err = db.Save(playbook).Error
		require.NoError(t, err)

		// 删除
		err = db.Delete(playbook).Error
		require.NoError(t, err)
	})
}

func TestScriptService_ScriptExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupIntegrationTestDB(t)
	log := logger.NewLogger("test", "debug")

	t.Run("Script lifecycle", func(t *testing.T) {
		script := &model.Script{
			Name:        "Test Script",
			Description: "Integration test script",
			Content:     "#!/bin/bash\necho 'Hello World'",
			Language:    "shell",
			Category:    "test",
			IsBuiltin:   false,
			IsActive:    true,
			Version:     1,
		}

		// 创建
		err := db.Create(script).Error
		require.NoError(t, err)
		assert.Greater(t, script.ID, uint(0))

		// 创建执行记录
		execution := &model.ScriptExecution{
			TaskID:       "test-task-123",
			ScriptID:     script.ID,
			ScriptName:   script.Name,
			ClusterName:  "test-cluster",
			TargetNodes:  `["node1", "node2"]`,
			Status:       "completed",
			SuccessCount: 2,
			FailedCount:  0,
			UserID:       1,
		}

		err = db.Create(execution).Error
		require.NoError(t, err)

		// 查询执行记录
		var retrieved model.ScriptExecution
		err = db.Where("task_id = ?", "test-task-123").First(&retrieved).Error
		require.NoError(t, err)
		assert.Equal(t, "completed", retrieved.Status)
		assert.Equal(t, 2, retrieved.SuccessCount)
	})
}

func TestWorkflowService_WorkflowExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupIntegrationTestDB(t)
	log := logger.NewLogger("test", "debug")

	t.Run("Workflow with multiple steps", func(t *testing.T) {
		// 创建简单的工作流定义
		workflowDef := `{
  "steps": [
    {
      "id": "step1",
      "name": "Check System",
      "type": "ssh",
      "action": "uptime",
      "timeout": 30
    },
    {
      "id": "step2",
      "name": "Collect Info",
      "type": "ssh",
      "action": "df -h",
      "depends_on": ["step1"],
      "timeout": 30
    }
  ]
}`

		workflow := &model.Workflow{
			Name:        "Test Workflow",
			Description: "Integration test workflow",
			Definition:  workflowDef,
			Category:    "test",
			IsBuiltin:   false,
			IsActive:    true,
			Version:     1,
		}

		// 创建工作流
		err := db.Create(workflow).Error
		require.NoError(t, err)
		assert.Greater(t, workflow.ID, uint(0))

		// 创建执行记录
		now := time.Now()
		execution := &model.WorkflowExecution{
			TaskID:       "workflow-test-123",
			WorkflowID:   workflow.ID,
			WorkflowName: workflow.Name,
			ClusterName:  "test-cluster",
			TargetNodes:  `["node1"]`,
			Status:       "running",
			CurrentStep:  "step1",
			StartTime:    &now,
			UserID:       1,
		}

		err = db.Create(execution).Error
		require.NoError(t, err)

		// 模拟完成
		endTime := time.Now()
		execution.Status = "completed"
		execution.EndTime = &endTime
		execution.Duration = int(endTime.Sub(now).Seconds())

		err = db.Save(execution).Error
		require.NoError(t, err)

		// 查询执行记录
		var retrieved model.WorkflowExecution
		err = db.Where("task_id = ?", "workflow-test-123").First(&retrieved).Error
		require.NoError(t, err)
		assert.Equal(t, "completed", retrieved.Status)
		assert.NotNil(t, retrieved.EndTime)
	})
}

func TestAutomation_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupIntegrationTestDB(t)
	log := logger.NewLogger("test", "debug")
	cm := automation.NewCredentialManager(db, log, "test-key-32-bytes-long-exactly")

	t.Run("Concurrent credential operations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 并发创建多个凭据
		numGoroutines := 10
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				credential := &model.SSHCredential{
					Name:     "concurrent-test-" + string(rune(index)),
					Username: "user" + string(rune(index)),
					AuthType: "password",
					Password: "password" + string(rune(index)),
					Port:     22,
				}
				err := cm.StoreCredential(credential)
				errChan <- err
			}(i)
		}

		// 收集结果
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-errChan:
				assert.NoError(t, err, "Concurrent operation failed")
			case <-ctx.Done():
				t.Fatal("Timeout waiting for concurrent operations")
			}
		}
	})
}

