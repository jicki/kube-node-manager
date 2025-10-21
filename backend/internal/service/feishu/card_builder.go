package feishu

import (
	"encoding/json"
	"fmt"
	"kube-node-manager/internal/service/k8s"
	"strings"
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
/node info <节点名> - 查看节点详情
/node cordon <节点名> [原因] - 禁止调度
/node uncordon <节点名> - 恢复调度节点

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

**其他命令**
/help - 显示此帮助信息
/help label - 标签管理帮助
/help taint - 污点管理帮助`,
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
