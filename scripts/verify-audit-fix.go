package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("🔍 验证audit日志外键约束修复")
	fmt.Println("==========================================")

	// 连接PostgreSQL
	dsn := "host=pgm-wz9lq79tmh67w5y4.pg.rds.aliyuncs.com port=5432 user=kube_node_mgr dbname=kube_node_mgr sslmode=disable password=3OBs4fb9CiHvMU5j"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("无法连接PostgreSQL:", err)
	}

	// 验证外键约束完整性
	fmt.Println("\n🔗 验证外键约束完整性:")

	var results []struct {
		Type        string
		Count       int64
		Description string
	}

	// 检查总记录数
	var totalCount int64
	db.Table("audit_logs").Count(&totalCount)
	results = append(results, struct {
		Type        string
		Count       int64
		Description string
	}{"total", totalCount, "总audit日志记录数"})

	// 检查有效的cluster_id记录
	var validClusterCount int64
	db.Table("audit_logs al").
		Joins("JOIN clusters c ON al.cluster_id = c.id").
		Count(&validClusterCount)
	results = append(results, struct {
		Type        string
		Count       int64
		Description string
	}{"valid_cluster", validClusterCount, "有效cluster_id的记录数"})

	// 检查cluster_id为NULL的记录
	var nullClusterCount int64
	db.Table("audit_logs").Where("cluster_id IS NULL").Count(&nullClusterCount)
	results = append(results, struct {
		Type        string
		Count       int64
		Description string
	}{"null_cluster", nullClusterCount, "cluster_id为NULL的记录数"})

	// 检查无效的cluster_id引用
	var invalidClusterCount int64
	db.Raw(`
		SELECT COUNT(*)
		FROM audit_logs al
		WHERE al.cluster_id IS NOT NULL 
		AND NOT EXISTS (SELECT 1 FROM clusters c WHERE c.id = al.cluster_id)
	`).Scan(&invalidClusterCount)
	results = append(results, struct {
		Type        string
		Count       int64
		Description string
	}{"invalid_cluster", invalidClusterCount, "无效cluster_id引用的记录数"})

	// 显示结果
	for _, result := range results {
		fmt.Printf("   %-15s: %6d  (%s)\n", result.Type, result.Count, result.Description)
	}

	// 验证最近的记录
	fmt.Println("\n📄 最近10条audit_logs记录:")
	var recentLogs []struct {
		ID           uint   `gorm:"column:id"`
		UserID       uint   `gorm:"column:user_id"`
		ClusterID    *uint  `gorm:"column:cluster_id"`
		ResourceType string `gorm:"column:resource_type"`
		Action       string `gorm:"column:action"`
		CreatedAt    string `gorm:"column:created_at"`
	}

	db.Table("audit_logs").
		Order("created_at DESC").
		Limit(10).
		Find(&recentLogs)

	for i, log := range recentLogs {
		clusterInfo := "NULL"
		if log.ClusterID != nil {
			clusterInfo = fmt.Sprintf("%d", *log.ClusterID)
		}
		fmt.Printf("   %2d. ID:%d UserID:%d ClusterID:%s Action:%s ResourceType:%s\n",
			i+1, log.ID, log.UserID, clusterInfo, log.Action, log.ResourceType)
	}

	// 按resource_type统计
	fmt.Println("\n📊 按resource_type统计:")
	var typeStats []struct {
		ResourceType string `gorm:"column:resource_type"`
		Count        int64  `gorm:"column:count"`
	}

	db.Table("audit_logs").
		Select("resource_type, COUNT(*) as count").
		Group("resource_type").
		Order("count DESC").
		Find(&typeStats)

	for _, stat := range typeStats {
		fmt.Printf("   %-20s: %d\n", stat.ResourceType, stat.Count)
	}

	// 评估修复效果
	fmt.Println("\n✅ 修复效果评估:")

	if invalidClusterCount == 0 {
		fmt.Println("   🎉 外键约束完整性: 完美 - 没有无效引用")
	} else {
		fmt.Printf("   ⚠️  外键约束完整性: 有问题 - %d个无效引用\n", invalidClusterCount)
	}

	clusterIntegrity := float64(validClusterCount+nullClusterCount) / float64(totalCount) * 100
	fmt.Printf("   📈 数据完整性: %.2f%% (%d/%d)\n", clusterIntegrity, validClusterCount+nullClusterCount, totalCount)

	if clusterIntegrity >= 99.0 {
		fmt.Println("   🌟 修复质量: 优秀")
	} else if clusterIntegrity >= 95.0 {
		fmt.Println("   👍 修复质量: 良好")
	} else {
		fmt.Println("   ⚠️  修复质量: 需要改进")
	}

	fmt.Println("\n🎉 验证完成！")
}
