package feishu

import (
	"encoding/json"
	"fmt"
)

// BuildErrorCard builds an error message card
func BuildErrorCard(errorMsg string) string {
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "red",
			"title": map[string]interface{}{
				"content": "❌ 错误",
				"tag":     "plain_text",
			},
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": errorMsg,
					"tag":     "plain_text",
				},
			},
		},
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildSuccessCard builds a success message card
func BuildSuccessCard(message string) string {
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "green",
			"title": map[string]interface{}{
				"content": "✅ 成功",
				"tag":     "plain_text",
			},
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": message,
					"tag":     "plain_text",
				},
			},
		},
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildHelpCard builds a help message card
func BuildHelpCard() string {
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "📖 机器人命令帮助",
				"tag":     "plain_text",
			},
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "markdown",
				"content": `**节点管理命令**
/node list [cluster] - 查询节点列表
/node info <cluster> <node_name> - 查询节点详情
/node cordon <cluster> <node_name> [reason] - 禁止调度节点
/node uncordon <cluster> <node_name> - 恢复调度节点

**集群管理命令**
/cluster list - 查询集群列表
/cluster status <cluster> - 查询集群状态

**审计日志命令**
/audit logs [user] [limit] - 查询审计日志（最多20条）

**其他命令**
/help - 显示此帮助信息`,
			},
			map[string]interface{}{
				"tag": "hr",
			},
			map[string]interface{}{
				"tag": "note",
				"elements": []interface{}{
					map[string]interface{}{
						"tag":     "plain_text",
						"content": "💡 提示：<cluster> 参数为必需，[参数] 为可选",
					},
				},
			},
		},
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildNodeListCard builds a node list card
func BuildNodeListCard(nodes []map[string]interface{}, clusterName string) string {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**集群**: %s\n**节点数量**: %d", clusterName, len(nodes)),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
	}

	// Add nodes
	for _, node := range nodes {
		status := "🟢 Ready"
		if ready, ok := node["ready"].(bool); ok && !ready {
			status = "🔴 NotReady"
		}

		schedulable := "✅ 可调度"
		if unschedulable, ok := node["unschedulable"].(bool); ok && unschedulable {
			schedulable = "⛔ 禁止调度"
		}

		nodeInfo := fmt.Sprintf("**%s**\n状态: %s | 调度: %s", node["name"], status, schedulable)

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": nodeInfo,
				"tag":     "lark_md",
			},
		})
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "📋 节点列表",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildNodeInfoCard builds a node info card
func BuildNodeInfoCard(node map[string]interface{}) string {
	status := "🟢 Ready"
	if ready, ok := node["ready"].(bool); ok && !ready {
		status = "🔴 NotReady"
	}

	schedulable := "✅ 可调度"
	if unschedulable, ok := node["unschedulable"].(bool); ok && unschedulable {
		schedulable = "⛔ 禁止调度"
	}

	content := fmt.Sprintf(`**节点名称**: %s
**状态**: %s
**调度状态**: %s
**IP 地址**: %s
**容器运行时**: %s
**内核版本**: %s
**操作系统**: %s`,
		node["name"],
		status,
		schedulable,
		node["internal_ip"],
		node["container_runtime"],
		node["kernel_version"],
		node["os_image"],
	)

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "🖥️ 节点详情",
				"tag":     "plain_text",
			},
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": content,
					"tag":     "lark_md",
				},
			},
		},
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildClusterListCard builds a cluster list card
func BuildClusterListCard(clusters []map[string]interface{}) string {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**集群数量**: %d", len(clusters)),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
	}

	// Add clusters
	for _, cluster := range clusters {
		status := "🟢 正常"
		if st, ok := cluster["status"].(string); ok && st != "active" {
			status = "🔴 异常"
		}

		clusterInfo := fmt.Sprintf("**%s**\n状态: %s | 节点数: %v",
			cluster["name"],
			status,
			cluster["node_count"],
		)

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": clusterInfo,
				"tag":     "lark_md",
			},
		})
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "🏢 集群列表",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildAuditLogsCard builds an audit logs card
func BuildAuditLogsCard(logs []map[string]interface{}) string {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**最近 %d 条审计日志**", len(logs)),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
	}

	// Add logs
	for _, log := range logs {
		status := "✅"
		if st, ok := log["status"].(string); ok && st != "success" {
			status = "❌"
		}

		logInfo := fmt.Sprintf("%s **%s** - %s\n操作: %s | 时间: %s",
			status,
			log["username"],
			log["details"],
			log["action"],
			log["created_at"],
		)

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": logInfo,
				"tag":     "lark_md",
			},
		})
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "📝 审计日志",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}
