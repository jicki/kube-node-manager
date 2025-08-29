package taint

import (
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/pkg/logger"
	"testing"
)

// 创建测试用的服务实例
func createTestService() *Service {
	return &Service{
		db:     nil, // 在这些测试中我们不需要数据库
		logger: logger.NewLogger(),
	}
}

func TestValidateTaints(t *testing.T) {
	service := createTestService()

	tests := []struct {
		name    string
		taints  []k8s.TaintInfo
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid taints - all non-empty values",
			taints: []k8s.TaintInfo{
				{Key: "key1", Value: "value1", Effect: "NoSchedule"},
				{Key: "key2", Value: "value2", Effect: "PreferNoSchedule"},
			},
			wantErr: false,
		},
		{
			name: "Valid taints - all empty values for same key",
			taints: []k8s.TaintInfo{
				{Key: "key1", Value: "", Effect: "NoSchedule"},
				{Key: "key1", Value: "", Effect: "NoExecute"},
			},
			wantErr: false,
		},
		{
			name: "Valid taints - different keys with mixed values",
			taints: []k8s.TaintInfo{
				{Key: "key1", Value: "", Effect: "NoSchedule"},
				{Key: "key2", Value: "value2", Effect: "NoExecute"},
			},
			wantErr: false,
		},
		{
			name: "Invalid taints - same key with mixed empty and non-empty values",
			taints: []k8s.TaintInfo{
				{Key: "key1", Value: "", Effect: "NoSchedule"},
				{Key: "key1", Value: "value1", Effect: "NoExecute"},
			},
			wantErr: true,
			errMsg:  "taint key 'key1': cannot have both empty and non-empty values simultaneously",
		},
		{
			name: "Invalid taints - multiple same keys with mixed values",
			taints: []k8s.TaintInfo{
				{Key: "key1", Value: "value1", Effect: "NoSchedule"},
				{Key: "key1", Value: "", Effect: "NoExecute"},
				{Key: "key1", Value: "value2", Effect: "PreferNoSchedule"},
			},
			wantErr: true,
			errMsg:  "taint key 'key1': cannot have both empty and non-empty values simultaneously",
		},
		{
			name: "Invalid taints - empty key",
			taints: []k8s.TaintInfo{
				{Key: "", Value: "value1", Effect: "NoSchedule"},
			},
			wantErr: true,
			errMsg:  "taint 1: key cannot be empty",
		},
		{
			name: "Invalid taints - invalid effect",
			taints: []k8s.TaintInfo{
				{Key: "key1", Value: "value1", Effect: "InvalidEffect"},
			},
			wantErr: true,
			errMsg:  "taint 1: invalid effect InvalidEffect, must be one of: NoSchedule, PreferNoSchedule, NoExecute",
		},
		{
			name: "Invalid taints - key with spaces",
			taints: []k8s.TaintInfo{
				{Key: "key with spaces", Value: "value1", Effect: "NoSchedule"},
			},
			wantErr: true,
			errMsg:  "taint 1: key cannot contain spaces",
		},
		{
			name: "Complex valid case - multiple keys with consistent values",
			taints: []k8s.TaintInfo{
				{Key: "key1", Value: "value1", Effect: "NoSchedule"},
				{Key: "key1", Value: "value2", Effect: "NoExecute"},
				{Key: "key2", Value: "", Effect: "NoSchedule"},
				{Key: "key2", Value: "", Effect: "PreferNoSchedule"},
				{Key: "key3", Value: "only-value", Effect: "NoExecute"},
			},
			wantErr: false,
		},
		{
			name: "Complex invalid case - one key violates rule among valid keys",
			taints: []k8s.TaintInfo{
				{Key: "key1", Value: "value1", Effect: "NoSchedule"},
				{Key: "key1", Value: "value2", Effect: "NoExecute"},
				{Key: "key2", Value: "", Effect: "NoSchedule"},
				{Key: "key2", Value: "mixed-value", Effect: "PreferNoSchedule"}, // This makes key2 invalid
				{Key: "key3", Value: "only-value", Effect: "NoExecute"},
			},
			wantErr: true,
			errMsg:  "taint key 'key2': cannot have both empty and non-empty values simultaneously",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateTaints(tt.taints)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateTaints() expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("validateTaints() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateTaints() unexpected error = %v", err)
				}
			}
		})
	}
}