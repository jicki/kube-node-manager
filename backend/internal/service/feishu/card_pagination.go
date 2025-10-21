package feishu

import (
	"encoding/json"
	"fmt"
	"math"
)

// PaginationConfig represents pagination configuration
type PaginationConfig struct {
	CurrentPage int
	PageSize    int
	TotalItems  int
	TotalPages  int
}

// BuildPaginatedNodeListCard builds a node list card with pagination
func BuildPaginatedNodeListCard(nodes []map[string]interface{}, clusterName string, pagination PaginationConfig) string {
	// Calculate pagination
	startIdx := (pagination.CurrentPage - 1) * pagination.PageSize
	endIdx := startIdx + pagination.PageSize
	if endIdx > len(nodes) {
		endIdx = len(nodes)
	}
	pageNodes := nodes[startIdx:endIdx]

	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**集群**: %s\n**节点数量**: %d | **页码**: %d/%d",
					clusterName, pagination.TotalItems, pagination.CurrentPage, pagination.TotalPages),
				"tag": "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
	}

	// Add nodes
	for _, node := range pageNodes {
		nodeName, _ := node["name"].(string)
		status := "🟢 Ready"
		if ready, ok := node["ready"].(bool); ok && !ready {
			status = "🔴 NotReady"
		}

		schedulable := "✅ 可调度"
		if unschedulable, ok := node["unschedulable"].(bool); ok && unschedulable {
			schedulable = "⛔ 禁止调度"
		}

		nodeInfo := fmt.Sprintf("**`%s`**\n状态: %s | 调度: %s", nodeName, status, schedulable)

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

	// Pagination buttons
	buttons := []interface{}{}

	// Previous page button
	if pagination.CurrentPage > 1 {
		buttons = append(buttons, map[string]interface{}{
			"tag": "button",
			"text": map[string]interface{}{
				"content": "⬅️ 上一页",
				"tag":     "plain_text",
			},
			"type": "default",
			"value": map[string]interface{}{
				"action":  "page_prev",
				"page":    pagination.CurrentPage - 1,
				"cluster": clusterName,
			},
		})
	}

	// Page indicator
	buttons = append(buttons, map[string]interface{}{
		"tag": "button",
		"text": map[string]interface{}{
			"content": fmt.Sprintf("%d/%d", pagination.CurrentPage, pagination.TotalPages),
			"tag":     "plain_text",
		},
		"type":     "default",
		"disabled": true,
	})

	// Next page button
	if pagination.CurrentPage < pagination.TotalPages {
		buttons = append(buttons, map[string]interface{}{
			"tag": "button",
			"text": map[string]interface{}{
				"content": "下一页 ➡️",
				"tag":     "plain_text",
			},
			"type": "default",
			"value": map[string]interface{}{
				"action":  "page_next",
				"page":    pagination.CurrentPage + 1,
				"cluster": clusterName,
			},
		})
	}

	if len(buttons) > 0 {
		elements = append(elements, map[string]interface{}{
			"tag":     "action",
			"actions": buttons,
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

// CalculatePagination calculates pagination parameters
func CalculatePagination(totalItems, currentPage, pageSize int) PaginationConfig {
	if pageSize <= 0 {
		pageSize = 10
	}
	if currentPage <= 0 {
		currentPage = 1
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	if currentPage > totalPages {
		currentPage = totalPages
	}

	return PaginationConfig{
		CurrentPage: currentPage,
		PageSize:    pageSize,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}
}

// BuildProgressCard builds a progress card for long-running operations
func BuildProgressCard(operation, target string, current, total int, status string) string {
	percentage := 0
	if total > 0 {
		percentage = (current * 100) / total
	}

	// Progress bar (simple text-based)
	progressBar := ""
	barLength := 20
	filledLength := (percentage * barLength) / 100
	for i := 0; i < barLength; i++ {
		if i < filledLength {
			progressBar += "█"
		} else {
			progressBar += "░"
		}
	}

	statusIcon := "⏳"
	if status == "completed" {
		statusIcon = "✅"
	} else if status == "failed" {
		statusIcon = "❌"
	}

	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**操作**: %s\n**目标**: %s", operation, target),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**进度**: %s %d%%\n\n`%s`\n\n%s %d/%d",
					statusIcon, percentage, progressBar, statusIcon, current, total),
				"tag": "lark_md",
			},
		},
	}

	if status != "completed" && status != "failed" {
		elements = append(elements, map[string]interface{}{
			"tag": "note",
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "plain_text",
					"content": "⏳ 操作进行中，请稍候...",
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
				"content": "📊 操作进度",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// BuildResourceUsageCard builds a card showing resource usage with progress bars
func BuildResourceUsageCard(nodeName string, cpuUsage, memoryUsage float64, cpuTotal, memoryTotal string) string {
	// CPU progress bar
	cpuBar := buildProgressBar(cpuUsage)
	memoryBar := buildProgressBar(memoryUsage)

	elements := []interface{}{
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**节点**: `%s`", nodeName),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**CPU 使用率**: %.1f%%\n`%s`\n总计: %s", cpuUsage, cpuBar, cpuTotal),
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**内存使用率**: %.1f%%\n`%s`\n总计: %s", memoryUsage, memoryBar, memoryTotal),
				"tag":     "lark_md",
			},
		},
	}

	// Warning for high usage
	if cpuUsage > 80 || memoryUsage > 80 {
		elements = append(elements, map[string]interface{}{
			"tag": "note",
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "plain_text",
					"content": "⚠️ 资源使用率较高，请关注",
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
				"content": "📈 资源使用情况",
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// buildProgressBar builds a text-based progress bar
func buildProgressBar(percentage float64) string {
	barLength := 20
	filledLength := int((percentage * float64(barLength)) / 100)

	bar := ""
	for i := 0; i < barLength; i++ {
		if i < filledLength {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return bar
}

// BuildTabCard builds a card with tab-like sections
func BuildTabCard(title string, tabs []TabSection, activeTab int) string {
	elements := []interface{}{}

	// Tab buttons
	tabButtons := []interface{}{}
	for i, tab := range tabs {
		buttonType := "default"
		if i == activeTab {
			buttonType = "primary"
		}

		tabButtons = append(tabButtons, map[string]interface{}{
			"tag": "button",
			"text": map[string]interface{}{
				"content": tab.Title,
				"tag":     "plain_text",
			},
			"type": buttonType,
			"value": map[string]interface{}{
				"action": "switch_tab",
				"tab":    i,
			},
		})
	}

	elements = append(elements, map[string]interface{}{
		"tag":     "action",
		"actions": tabButtons,
	})

	elements = append(elements, map[string]interface{}{
		"tag": "hr",
	})

	// Active tab content
	if activeTab >= 0 && activeTab < len(tabs) {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": tabs[activeTab].Content,
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
				"content": title,
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// TabSection represents a tab section
type TabSection struct {
	Title   string
	Content string
}
