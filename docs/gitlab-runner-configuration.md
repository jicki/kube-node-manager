# Gitlab 栏目 

* 系统配置 - 添加一个配置, 开启/关闭 Gitlab 配置. (开启 Gitlab 配置, 并配置 Gitlab domain, 认证 token. 显示 Gitlab 相关导航栏) 仅管理员.
* 如果 Gitlab 开启, 则左边导航栏显示 Gitlab 栏目. 
* 以新功能的形式添加，不能改动以及影响原来的功能以及模块.

## Runner 相关

https://docs.gitlab.com/api/runners/


## Pipeline 相关




---
  - ✅ 完全独立的新功能模块，不影响现有功能
  - ✅ GitLab 配置仅管理员可设置
  - ✅ 启用 GitLab 后，左侧导航栏动态显示 GitLab 菜单
  - ✅ 支持测试连接功能
  - ✅ Runners 页面支持类型和状态过滤
  - ✅ Pipelines 页面支持项目ID、分支、状态查询
  - ✅ 所有敏感信息（Token）妥善处理，不在前端暴露