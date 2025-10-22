package feishu

import (
	"encoding/json"
	"fmt"
)

// BuildNodeListCardWithActions builds a node list card with interactive action buttons
func BuildNodeListCardWithActions(nodes []map[string]interface{}, clusterName string) string {
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

	// Add nodes with action buttons
	for _, node := range nodes {
		nodeName, _ := node["name"].(string)
		status := "🟢 Ready"
		if ready, ok := node["ready"].(bool); ok && !ready {
			status = "🔴 NotReady"
		}

		schedulable := "✅ 可调度"
		unschedulable := false
		if u, ok := node["unschedulable"].(bool); ok && u {
			schedulable = "⛔ 禁止调度"
			unschedulable = true
		}

		// 获取节点类型（优先显示 user_type，其次是 roles）
		nodeType := ""
		if userType, ok := node["user_type"].(string); ok && userType != "" {
			nodeType = userType
		} else if roles, ok := node["roles"].(string); ok && roles != "" {
			nodeType = roles
		}

		// 节点信息 - 添加类型显示
		var nodeInfo string
		if nodeType != "" {
			nodeInfo = fmt.Sprintf("**`%s`**\n状态: %s | 调度: %s | 类型: %s", nodeName, status, schedulable, nodeType)
		} else {
			nodeInfo = fmt.Sprintf("**`%s`**\n状态: %s | 调度: %s", nodeName, status, schedulable)
		}

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": nodeInfo,
				"tag":     "lark_md",
			},
		})

		// 添加操作按钮
		buttons := []interface{}{}

		// 查看详情按钮（始终显示）
		buttons = append(buttons, map[string]interface{}{
			"tag": "button",
			"text": map[string]interface{}{
				"content": "📊 详情",
				"tag":     "plain_text",
			},
			"type": "primary",
			"value": map[string]interface{}{
				"action":  "node_info",
				"node":    nodeName,
				"cluster": clusterName,
			},
		})

		// 禁止/恢复调度按钮
		if unschedulable {
			buttons = append(buttons, map[string]interface{}{
				"tag": "button",
				"text": map[string]interface{}{
					"content": "✅ 恢复调度",
					"tag":     "plain_text",
				},
				"type": "default",
				"value": map[string]interface{}{
					"action":  "node_uncordon",
					"node":    nodeName,
					"cluster": clusterName,
				},
			})
		} else {
			buttons = append(buttons, map[string]interface{}{
				"tag": "button",
				"text": map[string]interface{}{
					"content": "⛔ 禁止调度",
					"tag":     "plain_text",
				},
				"type": "default",
				"value": map[string]interface{}{
					"action":  "node_cordon",
					"node":    nodeName,
					"cluster": clusterName,
				},
			})
		}

		elements = append(elements, map[string]interface{}{
			"tag":     "action",
			"actions": buttons,
		})

		elements = append(elements, map[string]interface{}{
			"tag": "hr",
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

// BuildNodeInfoCardWithActions builds a node info card with interactive buttons
func BuildNodeInfoCardWithActions(nodeInfo map[string]interface{}, clusterName string) string {
	nodeName, _ := nodeInfo["name"].(string)
	status, _ := nodeInfo["status"].(string)

	// Status icon
	statusIcon := "🟢"
	if status != "Ready" {
		statusIcon = "🔴"
	}

	unschedulable := false
	if u, ok := nodeInfo["unschedulable"].(bool); ok {
		unschedulable = u
	}

	schedulableText := "✅ 可调度"
	if unschedulable {
		schedulableText = "⛔ 禁止调度"
	}

	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**节点名称**: `%s`\n**集群**: %s", nodeName, clusterName),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**状态**: %s %s\n**调度**: %s", statusIcon, status, schedulableText),
				"tag":     "lark_md",
			},
		},
	}

	// 添加资源信息（如果有）
	if cpu, ok := nodeInfo["cpu"].(string); ok {
		memory, _ := nodeInfo["memory"].(string)
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**资源**\nCPU: %s\n内存: %s", cpu, memory),
				"tag":     "lark_md",
			},
		})
	}

	elements = append(elements, map[string]interface{}{
		"tag": "hr",
	})

	// 操作按钮
	buttons := []interface{}{
		map[string]interface{}{
			"tag": "button",
			"text": map[string]interface{}{
				"content": "🔄 刷新",
				"tag":     "plain_text",
			},
			"type": "default",
			"value": map[string]interface{}{
				"action":  "node_refresh",
				"node":    nodeName,
				"cluster": clusterName,
			},
		},
	}

	// 禁止/恢复调度按钮
	if unschedulable {
		buttons = append(buttons, map[string]interface{}{
			"tag": "button",
			"text": map[string]interface{}{
				"content": "✅ 恢复调度",
				"tag":     "plain_text",
			},
			"type": "primary",
			"value": map[string]interface{}{
				"action":  "node_uncordon",
				"node":    nodeName,
				"cluster": clusterName,
			},
		})
	} else {
		buttons = append(buttons, map[string]interface{}{
			"tag": "button",
			"text": map[string]interface{}{
				"content": "⛔ 禁止调度",
				"tag":     "plain_text",
			},
			"type": "danger",
			"value": map[string]interface{}{
				"action":  "node_cordon",
				"node":    nodeName,
				"cluster": clusterName,
			},
		})
	}

	elements = append(elements, map[string]interface{}{
		"tag":     "action",
		"actions": buttons,
	})

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "📊 节点详情",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildClusterListCardWithActions builds a cluster list card with switch buttons
func BuildClusterListCardWithActions(clusters []map[string]interface{}, currentCluster string) string {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**集群数量**: %d\n**当前集群**: %s", len(clusters), currentCluster),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
	}

	// Add clusters with action buttons
	for _, cluster := range clusters {
		clusterName, _ := cluster["name"].(string)
		status := "🟢 健康"
		if s, ok := cluster["status"].(string); ok && s != "Healthy" {
			status = "⚠️ " + s
		}

		// 获取节点数量
		nodeCount := 0
		if n, ok := cluster["nodes"].(int); ok {
			nodeCount = n
		}

		// 获取集群版本
		version := "未知"
		if v, ok := cluster["version"].(string); ok && v != "" {
			version = v
		}

		isCurrent := clusterName == currentCluster
		clusterPrefix := ""
		if isCurrent {
			clusterPrefix = "👉 "
		}

		clusterInfo := fmt.Sprintf("%s**%s**\n状态: %s | 节点: %d | 版本: %s", clusterPrefix, clusterName, status, nodeCount, version)

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": clusterInfo,
				"tag":     "lark_md",
			},
		})

		// 添加切换按钮（非当前集群）
		if !isCurrent {
			buttons := []interface{}{
				map[string]interface{}{
					"tag": "button",
					"text": map[string]interface{}{
						"content": "🔄 切换",
						"tag":     "plain_text",
					},
					"type": "primary",
					"value": map[string]interface{}{
						"action":  "cluster_switch",
						"cluster": clusterName,
					},
				},
				map[string]interface{}{
					"tag": "button",
					"text": map[string]interface{}{
						"content": "📊 状态",
						"tag":     "plain_text",
					},
					"type": "default",
					"value": map[string]interface{}{
						"action":  "cluster_status",
						"cluster": clusterName,
					},
				},
			}

			elements = append(elements, map[string]interface{}{
				"tag":     "action",
				"actions": buttons,
			})
		}

		elements = append(elements, map[string]interface{}{
			"tag": "hr",
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

// BuildConfirmActionCard builds a confirmation card for dangerous operations
func BuildConfirmActionCard(action, target, description, confirmCommand string) string {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**操作**: %s\n**目标**: `%s`", action, target),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**说明**\n\n%s", description),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag": "note",
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "plain_text",
					"content": "⚠️ 此操作需要确认。请仔细核对后再点击确认。",
				},
			},
		},
	}

	// 确认和取消按钮
	buttons := []interface{}{
		map[string]interface{}{
			"tag": "button",
			"text": map[string]interface{}{
				"content": "✅ 确认执行",
				"tag":     "plain_text",
			},
			"type": "danger",
			"value": map[string]interface{}{
				"action":  "confirm_action",
				"command": confirmCommand,
			},
		},
		map[string]interface{}{
			"tag": "button",
			"text": map[string]interface{}{
				"content": "❌ 取消",
				"tag":     "plain_text",
			},
			"type": "default",
			"value": map[string]interface{}{
				"action": "cancel_action",
			},
		},
	}

	elements = append(elements, map[string]interface{}{
		"tag":     "action",
		"actions": buttons,
	})

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "orange",
			"title": map[string]interface{}{
				"content": "⚠️ 操作确认",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}
