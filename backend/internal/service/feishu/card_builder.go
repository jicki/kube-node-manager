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
				"content": `**集群管理命令**
/cluster list - 查看所有集群列表
/cluster status <集群名> - 查看集群状态

**节点管理命令**
/node set <集群名> - 切换到指定集群
/node list - 查看当前集群的节点列表
/node info <节点名> - 查看节点详情
/node cordon <节点名> - 禁止调度
/node cordon <节点名> <禁止调度说明> - 禁止调度
/node uncordon <节点名> - 恢复调度节点

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
						"content": "💡 提示：需要先使用 /cluster list 查看集群，然后使用 /node set 选择集群，最后使用 /node list 查看节点",
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

		// 处理节点类型 - 优先使用 deeproute.cn/user-type 标签
		roleText := ""
		if userType, ok := node["user_type"].(string); ok && userType != "" {
			// 使用 deeproute.cn/user-type 标签值
			roleIcons := map[string]string{
				"gpu":     "🎮",
				"cpu":     "💻",
				"storage": "💾",
				"network": "🌐",
				"master":  "👑",
			}
			icon := roleIcons[userType]
			if icon == "" {
				icon = "📌"
			}
			roleText = fmt.Sprintf("%s %s", icon, userType)
		} else if roles, ok := node["roles"].([]string); ok && len(roles) > 0 {
			// 回退到使用 roles
			roleIcons := map[string]string{
				"master":        "👑",
				"control-plane": "👑",
				"worker":        "⚙️",
			}
			for _, role := range roles {
				icon := roleIcons[role]
				if icon == "" {
					icon = "📌"
				}
				if roleText != "" {
					roleText += " "
				}
				roleText += fmt.Sprintf("%s %s", icon, role)
			}
		} else {
			roleText = "⚙️ worker"
		}

		nodeInfo := fmt.Sprintf("**%s**\n类型: %s\n状态: %s | 调度: %s", node["name"], roleText, status, schedulable)

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
		name := cluster["name"].(string)
		status := cluster["status"].(string)
		nodes := cluster["nodes"]

		clusterInfo := fmt.Sprintf("**📦 %s**\n状态: %s | 节点数: %v\n\n💡 使用命令切换: `/node set %s`",
			name,
			status,
			nodes,
			name,
		)

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": clusterInfo,
				"tag":     "lark_md",
			},
		})
	}

	// Add usage note
	elements = append(elements, map[string]interface{}{
		"tag": "hr",
	})
	elements = append(elements, map[string]interface{}{
		"tag": "note",
		"elements": []interface{}{
			map[string]interface{}{
				"tag":     "plain_text",
				"content": "💡 使用 /node set <集群名> 切换到指定集群后，使用 /node list 查看节点",
			},
		},
	})

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

// BuildClusterStatusCard builds a cluster status card
func BuildClusterStatusCard(name, statusIcon, statusText string, totalNodes, healthyNodes, unhealthyNodes int) string {
	content := fmt.Sprintf(`**集群**: %s
**状态**: %s %s
**节点数**: %d
**健康节点**: %d
**不健康节点**: %d`,
		name,
		statusIcon,
		statusText,
		totalNodes,
		healthyNodes,
		unhealthyNodes,
	)

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "🏢 集群状态",
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

// BuildCordonHelpCard 构建禁止调度帮助卡片
func BuildCordonHelpCard() string {
	elements := []interface{}{
		// 用法说明
		map[string]interface{}{
			"tag":     "markdown",
			"content": "**📋 用法**\n```\n/node cordon <节点名> [原因]\n```",
		},
		map[string]interface{}{
			"tag": "hr",
		},
		// 常用原因
		map[string]interface{}{
			"tag":     "markdown",
			"content": "**🔖 常用原因**（可直接复制使用）",
		},
		map[string]interface{}{
			"tag": "div",
			"fields": []interface{}{
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "🔧 **维护**\n`/node cordon <节点名> 维护`",
					},
				},
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "⬆️ **升级**\n`/node cordon <节点名> 升级`",
					},
				},
			},
		},
		map[string]interface{}{
			"tag": "div",
			"fields": []interface{}{
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "🔍 **故障排查**\n`/node cordon <节点名> 故障排查`",
					},
				},
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "⚠️ **资源不足**\n`/node cordon <节点名> 资源不足`",
					},
				},
			},
		},
		map[string]interface{}{
			"tag": "div",
			"fields": []interface{}{
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "🔄 **重启**\n`/node cordon <节点名> 重启`",
					},
				},
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "🧪 **测试**\n`/node cordon <节点名> 测试`",
					},
				},
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		// 示例
		map[string]interface{}{
			"tag":     "markdown",
			"content": "**📝 示例**\n```\n/node cordon 10-9-9-28.vm.pd.sz.deeproute.ai 维护升级\n```",
		},
		map[string]interface{}{
			"tag": "note",
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "plain_text",
					"content": "💡 提示：原因可选，但建议填写以便团队协作",
				},
			},
		},
	}

	config := map[string]interface{}{
		"wide_screen_mode": true,
	}

	header := map[string]interface{}{
		"template": "blue",
		"title": map[string]interface{}{
			"content": "💡 节点禁止调度指南",
			"tag":     "plain_text",
		},
	}

	card := map[string]interface{}{
		"config":   config,
		"header":   header,
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}
