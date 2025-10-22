package feishu

import (
	"encoding/json"
	"fmt"
	"kube-node-manager/internal/service/k8s"
	"strings"
	"time"
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

// BuildEnhancedErrorCard builds an enhanced error card with code, suggestion, and details
func BuildEnhancedErrorCard(code, message, suggestion, details string) string {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**错误**: %s", message),
				"tag":     "lark_md",
			},
		},
	}

	// 添加错误码（如果有）
	if code != "" {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**错误码**: `%s`", code),
				"tag":     "lark_md",
			},
		})
	}

	// 添加建议（如果有）
	if suggestion != "" {
		elements = append(elements, map[string]interface{}{
			"tag": "hr",
		})
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**💡 解决建议**",
				"tag":     "lark_md",
			},
		})
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": suggestion,
				"tag":     "lark_md",
			},
		})
	}

	// 添加技术详情（如果有，用于调试）
	if details != "" {
		elements = append(elements, map[string]interface{}{
			"tag": "hr",
		})
		elements = append(elements, map[string]interface{}{
			"tag": "note",
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "plain_text",
					"content": fmt.Sprintf("技术详情: %s", details),
				},
			},
		})
	}

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
		"elements": elements,
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
/cluster set <集群名> - 切换到指定集群
/cluster status <集群名> - 查看集群状态

**节点管理命令**
/node list - 查看当前集群的节点列表
/node list <关键词> - 模糊搜索节点（如: /node list 10-3）
/node info <节点名> - 查看节点详情
/node cordon <节点名> [原因] - 禁止调度
/node uncordon <节点名> - 恢复调度节点
/node batch <operation> <nodes> - 批量操作

**标签管理命令**
/label list <节点名> - 查看节点标签
/label add <节点名> <key>=<value> - 添加标签
/label remove <节点名> <key> - 删除标签

**污点管理命令**
/taint list <节点名> - 查看节点污点
/taint add <节点名> <key>=<value>:<effect> - 添加污点
/taint remove <节点名> <key> - 删除污点

**审计日志命令**
/audit logs [user] [limit] - 查询审计日志（最多20条）

**快捷命令**
/quick status - 当前集群概览
/quick nodes - 显示问题节点
/quick health - 所有集群健康检查

**其他命令**
/help - 显示此帮助信息
/help label - 标签管理帮助
/help taint - 污点管理帮助
/help batch - 批量操作帮助
/help quick - 快捷命令帮助`,
			},
			map[string]interface{}{
				"tag": "hr",
			},
			map[string]interface{}{
				"tag": "note",
				"elements": []interface{}{
					map[string]interface{}{
						"tag":     "plain_text",
						"content": "💡 提示：需要先使用 /cluster list 查看集群，然后使用 /cluster set 选择集群，最后使用 /node list 查看节点",
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

		// 使用代码块格式避免节点名称被识别为超链接
		nodeInfo := fmt.Sprintf("**`%s`**\n类型: %s\n状态: %s | 调度: %s", node["name"], roleText, status, schedulable)

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

	// 使用代码块格式避免节点名称被识别为超链接
	content := fmt.Sprintf("**节点名称**: `%s`\n**状态**: %s\n**调度状态**: %s\n**IP 地址**: %s\n**容器运行时**: %s\n**内核版本**: %s\n**操作系统**: %s",
		node["name"],
		status,
		schedulable,
		node["internal_ip"],
		node["container_runtime"],
		node["kernel_version"],
		node["os_image"],
	)

	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": content,
				"tag":     "lark_md",
			},
		},
	}

	// 添加资源显示
	if capacity, ok := node["capacity"].(map[string]interface{}); ok {
		if allocatable, ok := node["allocatable"].(map[string]interface{}); ok {
			// 添加分隔线
			elements = append(elements, map[string]interface{}{
				"tag": "hr",
			})

			// 添加资源标题
			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**💾 资源显示**",
					"tag":     "lark_md",
				},
			})

			// 添加资源说明
			elements = append(elements, map[string]interface{}{
				"tag": "note",
				"elements": []interface{}{
					map[string]interface{}{
						"tag":     "plain_text",
						"content": "总量 / 可分配 / 使用量",
					},
				},
			})

			// CPU
			cpuCapacity := getStringValue(capacity, "cpu")
			cpuAllocatable := getStringValue(allocatable, "cpu")
			cpuUsage := getStringValue(node, "cpu_usage")
			if cpuUsage == "" {
				cpuUsage = "N/A"
			}

			// Memory
			memCapacity := getStringValue(capacity, "memory")
			memAllocatable := getStringValue(allocatable, "memory")
			memUsage := getStringValue(node, "memory_usage")
			if memUsage == "" {
				memUsage = "N/A"
			}

			// Pods
			podsCapacity := getStringValue(capacity, "pods")
			podsAllocatable := getStringValue(allocatable, "pods")

			// GPU
			gpuCapacity := "0"
			gpuAllocatable := "0"
			if gpuMap, ok := capacity["gpu"].(map[string]interface{}); ok && len(gpuMap) > 0 {
				for _, v := range gpuMap {
					if val, ok := v.(string); ok {
						gpuCapacity = val
						break
					}
				}
			}
			if gpuMap, ok := allocatable["gpu"].(map[string]interface{}); ok && len(gpuMap) > 0 {
				for _, v := range gpuMap {
					if val, ok := v.(string); ok {
						gpuAllocatable = val
						break
					}
				}
			}

			resourceContent := fmt.Sprintf(`🟢 **CPU**: %s / %s / %s
🔵 **内存**: %s / %s / %s
🟣 **POD**: %s / %s / N/A
🔴 **GPU**: %s / %s / N/A`,
				cpuCapacity, cpuAllocatable, cpuUsage,
				memCapacity, memAllocatable, memUsage,
				podsCapacity, podsAllocatable,
				gpuCapacity, gpuAllocatable,
			)

			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": resourceContent,
					"tag":     "lark_md",
				},
			})
		}
	}

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
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// getStringValue 辅助函数，从 map 中获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
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

// BuildLabelListCard builds a label list card
func BuildLabelListCard(labels map[string]string, nodeName, clusterName string) string {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**节点**: `%s`\n**集群**: %s\n**标签数量**: %d", nodeName, clusterName, len(labels)),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
	}

	if len(labels) == 0 {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "该节点没有自定义标签",
				"tag":     "plain_text",
			},
		})
	} else {
		// 分类显示标签
		systemLabels := make(map[string]string)
		userLabels := make(map[string]string)

		systemPrefixes := []string{
			"kubernetes.io/",
			"k8s.io/",
			"node.kubernetes.io/",
			"node-role.kubernetes.io/",
			"beta.kubernetes.io/",
			"topology.kubernetes.io/",
		}

		for key, value := range labels {
			isSystem := false
			for _, prefix := range systemPrefixes {
				if strings.HasPrefix(key, prefix) {
					isSystem = true
					break
				}
			}
			if isSystem {
				systemLabels[key] = value
			} else {
				userLabels[key] = value
			}
		}

		// 显示用户标签
		if len(userLabels) > 0 {
			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**🏷️ 用户标签**",
					"tag":     "lark_md",
				},
			})

			labelTexts := make([]string, 0, len(userLabels))
			for key, value := range userLabels {
				labelTexts = append(labelTexts, fmt.Sprintf("• `%s` = `%s`", key, value))
			}

			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": strings.Join(labelTexts, "\n"),
					"tag":     "lark_md",
				},
			})
		}

		// 显示系统标签（折叠显示前5个）
		if len(systemLabels) > 0 {
			elements = append(elements, map[string]interface{}{
				"tag": "hr",
			})
			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": fmt.Sprintf("**⚙️ 系统标签** (%d 个)", len(systemLabels)),
					"tag":     "lark_md",
				},
			})

			labelTexts := make([]string, 0)
			count := 0
			for key, value := range systemLabels {
				if count < 5 {
					labelTexts = append(labelTexts, fmt.Sprintf("• `%s` = `%s`", key, value))
					count++
				}
			}

			if len(systemLabels) > 5 {
				labelTexts = append(labelTexts, fmt.Sprintf("... 还有 %d 个系统标签", len(systemLabels)-5))
			}

			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": strings.Join(labelTexts, "\n"),
					"tag":     "lark_md",
				},
			})
		}
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "🏷️ 节点标签列表",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildLabelHelpCard builds a label help card
func BuildLabelHelpCard() string {
	elements := []interface{}{
		map[string]interface{}{
			"tag":     "markdown",
			"content": "**📋 用法**\n```\n/label add <节点名> <key>=<value>\n/label remove <节点名> <key>\n/label list <节点名>\n```",
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag":     "markdown",
			"content": "**📝 示例**",
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": "**添加单个标签**\n```\n/label add node-1 env=production\n```",
			},
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": "**添加多个标签**\n```\n/label add node-1 env=prod,app=web,version=v1.0\n```",
			},
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": "**删除标签**\n```\n/label remove node-1 env\n```",
			},
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": "**查看标签**\n```\n/label list node-1\n```",
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
					"content": "💡 提示：标签 key 和 value 必须符合 Kubernetes 命名规范",
				},
			},
		},
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "💡 标签管理指南",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildTaintListCard builds a taint list card
func BuildTaintListCard(taints []k8s.TaintInfo, nodeName, clusterName string) string {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**节点**: `%s`\n**集群**: %s\n**污点数量**: %d", nodeName, clusterName, len(taints)),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
	}

	if len(taints) == 0 {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "该节点没有污点",
				"tag":     "plain_text",
			},
		})
	} else {
		for i, taint := range taints {
			effectIcon := "⚠️"
			effectDesc := ""
			switch taint.Effect {
			case "NoSchedule":
				effectIcon = "⛔"
				effectDesc = "不调度新 Pod"
			case "PreferNoSchedule":
				effectIcon = "⚠️"
				effectDesc = "尽量不调度新 Pod"
			case "NoExecute":
				effectIcon = "🚫"
				effectDesc = "驱逐现有 Pod"
			}

			taintText := fmt.Sprintf("**%s Taint %d**\n• Key: `%s`\n• Value: `%s`\n• Effect: %s %s (%s)",
				effectIcon, i+1, taint.Key, taint.Value, effectIcon, taint.Effect, effectDesc)

			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": taintText,
					"tag":     "lark_md",
				},
			})

			if i < len(taints)-1 {
				elements = append(elements, map[string]interface{}{
					"tag": "hr",
				})
			}
		}
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "🏷️ 节点污点列表",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildTaintHelpCard builds a taint help card
func BuildTaintHelpCard() string {
	elements := []interface{}{
		map[string]interface{}{
			"tag":     "markdown",
			"content": "**📋 用法**\n```\n/taint add <节点名> <key>=<value>:<effect>\n/taint remove <节点名> <key>\n/taint list <节点名>\n```",
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag":     "markdown",
			"content": "**🔖 Effect 类型**",
		},
		map[string]interface{}{
			"tag": "div",
			"fields": []interface{}{
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "**⛔ NoSchedule**\n不调度新 Pod",
					},
				},
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "**⚠️ PreferNoSchedule**\n尽量不调度",
					},
				},
			},
		},
		map[string]interface{}{
			"tag": "div",
			"fields": []interface{}{
				map[string]interface{}{
					"is_short": false,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": "**🚫 NoExecute**\n不调度且驱逐现有 Pod（危险操作）",
					},
				},
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag":     "markdown",
			"content": "**📝 示例**",
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": "**添加污点**\n```\n/taint add node-1 key1=value1:NoSchedule\n```",
			},
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": "**删除污点**\n```\n/taint remove node-1 key1\n```",
			},
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": "**查看污点**\n```\n/taint list node-1\n```",
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
					"content": "💡 提示：NoExecute 会立即驱逐节点上的 Pod，请谨慎使用",
				},
			},
		},
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "💡 污点管理指南",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildTaintNoExecuteWarningCard builds a warning card for NoExecute taint
func BuildTaintNoExecuteWarningCard(nodeName string, taints []k8s.TaintInfo) string {
	taintStrs := make([]string, 0, len(taints))
	for _, t := range taints {
		taintStrs = append(taintStrs, fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Effect))
	}

	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**节点**: `%s`\n**污点**: %s", nodeName, strings.Join(taintStrs, ", ")),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**⚠️ 警告**\n\nNoExecute 污点会立即驱逐节点上所有不能容忍该污点的 Pod，这可能导致服务中断。\n\n请确认您了解此操作的影响。",
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "note",
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "plain_text",
					"content": "💡 如需继续，请联系管理员通过 Web 界面操作",
				},
			},
		},
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "red",
			"title": map[string]interface{}{
				"content": "⚠️ 危险操作确认",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildBatchHelpCard builds a help card for batch operations
func BuildBatchHelpCard() string {
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "📋 批量操作命令帮助",
				"tag":     "plain_text",
			},
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**命令格式**",
					"tag":     "lark_md",
				},
			},
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "`/node batch <operation> <node1,node2,node3> [args...]`",
					"tag":     "lark_md",
				},
			},
			map[string]interface{}{
				"tag": "hr",
			},
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**支持的操作**",
					"tag":     "lark_md",
				},
			},
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "• `cordon` - 批量禁止调度\n• `uncordon` - 批量恢复调度",
					"tag":     "lark_md",
				},
			},
			map[string]interface{}{
				"tag": "hr",
			},
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**使用示例**",
					"tag":     "lark_md",
				},
			},
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "批量禁止调度:\n`/node batch cordon node-1,node-2,node-3 维护中`\n\n批量恢复调度:\n`/node batch uncordon node-1,node-2,node-3`",
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
						"content": "💡 提示：节点名称之间用逗号分隔，不要有空格",
					},
				},
			},
		},
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildBatchOperationResultCard builds a result card for batch operations
func BuildBatchOperationResultCard(operation, clusterName string, nodeNames []string, results map[string]string, successCount, failureCount int, reason string) string {
	// 确定卡片颜色和标题
	cardTemplate := "green"
	titlePrefix := "✅"
	if failureCount > 0 {
		if successCount == 0 {
			cardTemplate = "red"
			titlePrefix = "❌"
		} else {
			cardTemplate = "orange"
			titlePrefix = "⚠️"
		}
	}

	// 构建元素列表
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**操作**: %s\n**集群**: %s\n**总计**: %d 个节点", operation, clusterName, len(nodeNames)),
				"tag":     "lark_md",
			},
		},
	}

	// 添加原因（如果有）
	if reason != "" {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**原因**: %s", reason),
				"tag":     "lark_md",
			},
		})
	}

	elements = append(elements, map[string]interface{}{
		"tag": "hr",
	})

	// 统计结果
	elements = append(elements, map[string]interface{}{
		"tag": "div",
		"text": map[string]interface{}{
			"content": fmt.Sprintf("**执行结果**\n\n✅ 成功: %d 个\n❌ 失败: %d 个", successCount, failureCount),
			"tag":     "lark_md",
		},
	})

	// 如果有失败的节点，显示详情
	if failureCount > 0 {
		elements = append(elements, map[string]interface{}{
			"tag": "hr",
		})
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**失败详情**",
				"tag":     "lark_md",
			},
		})

		// 构建失败节点列表
		var failedNodes []string
		for _, nodeName := range nodeNames {
			if result, ok := results[nodeName]; ok && result != "success" {
				failedNodes = append(failedNodes, fmt.Sprintf("• `%s`: %s", nodeName, result))
			}
		}

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": strings.Join(failedNodes, "\n"),
				"tag":     "lark_md",
			},
		})
	}

	// 成功的节点列表（如果有）
	if successCount > 0 && successCount <= 10 { // 只显示前10个成功节点
		elements = append(elements, map[string]interface{}{
			"tag": "hr",
		})
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**成功的节点**",
				"tag":     "lark_md",
			},
		})

		var successNodes []string
		for _, nodeName := range nodeNames {
			if result, ok := results[nodeName]; ok && result == "success" {
				successNodes = append(successNodes, fmt.Sprintf("`%s`", nodeName))
			}
		}

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": strings.Join(successNodes, ", "),
				"tag":     "lark_md",
			},
		})
	} else if successCount > 10 {
		elements = append(elements, map[string]interface{}{
			"tag": "note",
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "plain_text",
					"content": fmt.Sprintf("成功节点较多（%d个），已省略显示", successCount),
				},
			},
		})
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": cardTemplate,
			"title": map[string]interface{}{
				"content": fmt.Sprintf("%s 批量%s完成", titlePrefix, operation),
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildQuickHelpCard builds a help card for quick commands
func BuildQuickHelpCard() string {
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "⚡ 快捷命令帮助",
				"tag":     "plain_text",
			},
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**可用的快捷命令**",
					"tag":     "lark_md",
				},
			},
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "• `/quick status` - 当前集群概览\n• `/quick nodes` - 显示问题节点（NotReady/禁止调度）\n• `/quick health` - 所有集群健康检查",
					"tag":     "lark_md",
				},
			},
			map[string]interface{}{
				"tag": "hr",
			},
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "**使用示例**",
					"tag":     "lark_md",
				},
			},
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": "查看当前集群状态:\n`/quick status`\n\n查看问题节点:\n`/quick nodes`\n\n检查所有集群健康状态:\n`/quick health`",
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
						"content": "💡 提示：快捷命令会自动聚合常用信息，提供快速概览",
					},
				},
			},
		},
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildQuickStatusCard builds a status card for quick status command
func BuildQuickStatusCard(clusterName string, statusData interface{}, totalNodes, readyNodes, notReadyNodes, unschedulableNodes int) string {
	// Parse status data - using simple string formatting
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**集群**: %s", clusterName),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**节点统计**",
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("• 总节点数: %d\n• Ready: %d\n• NotReady: %d\n• 禁止调度: %d", totalNodes, readyNodes, notReadyNodes, unschedulableNodes),
				"tag":     "lark_md",
			},
		},
	}

	// Warning if there are problematic nodes
	if notReadyNodes > 0 || unschedulableNodes > 0 {
		elements = append(elements, map[string]interface{}{
			"tag": "note",
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "plain_text",
					"content": "⚠️ 发现问题节点，建议使用 /quick nodes 查看详情",
				},
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
				"content": "⚡ 集群快速状态",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildQuickNodesCard builds a card showing problematic nodes
func BuildQuickNodesCard(clusterName string, nodes interface{}) string {
	// Type assertion for k8s.NodeInfo slice
	var nodeList []k8s.NodeInfo
	var nodeCount int

	if nodeSlice, ok := nodes.([]k8s.NodeInfo); ok {
		nodeList = nodeSlice
		nodeCount = len(nodeSlice)
	}

	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**集群**: %s\n**问题节点数**: %d", clusterName, nodeCount),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
	}

	if nodeCount == 0 {
		elements = append(elements, map[string]interface{}{
			"tag": "note",
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "plain_text",
					"content": "✅ 太好了！当前没有问题节点",
				},
			},
		})
	} else {
		// 显示问题节点列表
		for _, n := range nodeList {
			// 判断节点是否 Ready（状态应该是 "Ready" 或 "Ready,xxx"，而不是 "NotReady"）
			isReady := strings.HasPrefix(n.Status, "Ready,") || n.Status == "Ready"
			status := "🟢 Ready"
			if !isReady {
				status = "🔴 NotReady"
			}

			schedulable := "✅ 可调度"
			if !n.Schedulable {
				schedulable = "⛔ 禁止调度"
				// 如果有禁止调度原因，添加原因
				if n.UnschedulableReason != "" {
					schedulable = fmt.Sprintf("⛔ 禁止调度（%s）", n.UnschedulableReason)
				}
			}

			// 获取异常开始时间（仅针对真正 NotReady 的节点）
			var abnormalSince string
			if !isReady {
				// 从 Conditions 中查找 Ready 状态的 LastTransitionTime
				for _, cond := range n.Conditions {
					if cond.Type == "Ready" && cond.Status != "True" {
						duration := time.Since(cond.LastTransitionTime)
						if duration < time.Minute {
							abnormalSince = fmt.Sprintf("%.0f秒", duration.Seconds())
						} else if duration < time.Hour {
							abnormalSince = fmt.Sprintf("%.0f分钟", duration.Minutes())
						} else if duration < 24*time.Hour {
							abnormalSince = fmt.Sprintf("%.1f小时", duration.Hours())
						} else {
							abnormalSince = fmt.Sprintf("%.1f天", duration.Hours()/24)
						}
						break
					}
				}
			}

			// 构建节点信息
			var nodeInfo string
			if abnormalSince != "" {
				nodeInfo = fmt.Sprintf("**`%s`**\n状态: %s | 调度: %s | 异常时长: %s", n.Name, status, schedulable, abnormalSince)
			} else {
				nodeInfo = fmt.Sprintf("**`%s`**\n状态: %s | 调度: %s", n.Name, status, schedulable)
			}

			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": nodeInfo,
					"tag":     "lark_md",
				},
			})
			elements = append(elements, map[string]interface{}{
				"tag": "hr",
			})
		}
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "orange",
			"title": map[string]interface{}{
				"content": "⚠️ 问题节点",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildQuickHealthCard builds a health check card for all clusters
func BuildQuickHealthCard(healthData interface{}) string {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**所有集群健康状态**",
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
	}

	// This is simplified; in real implementation, properly parse healthData
	elements = append(elements, map[string]interface{}{
		"tag": "div",
		"text": map[string]interface{}{
			"content": "集群健康检查已完成。详细信息请使用 /cluster list 和 /cluster status 查看。",
			"tag":     "lark_md",
		},
	})

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": "⚡ 集群健康检查",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}
