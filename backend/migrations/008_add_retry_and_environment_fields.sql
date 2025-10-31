-- 添加重试策略、环境标签和风险等级字段
-- 支持任务失败自动重试和环境保护

-- ==========================================
-- 1. 为 ansible_tasks 表添加重试相关字段
-- ==========================================

-- 添加重试策略字段
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS retry_policy JSONB;
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS max_retries INTEGER DEFAULT 0;

-- 添加注释
COMMENT ON COLUMN ansible_tasks.retry_policy IS '重试策略(JSON)';
COMMENT ON COLUMN ansible_tasks.retry_count IS '当前重试次数';
COMMENT ON COLUMN ansible_tasks.max_retries IS '最大重试次数';

-- ==========================================
-- 2. 为 ansible_inventories 表添加环境标签字段
-- ==========================================

-- 添加环境标签字段
ALTER TABLE ansible_inventories ADD COLUMN IF NOT EXISTS environment VARCHAR(20) DEFAULT 'dev';

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_ansible_inventories_environment ON ansible_inventories(environment);

-- 添加注释
COMMENT ON COLUMN ansible_inventories.environment IS '环境标签(dev/staging/production)';

-- ==========================================
-- 3. 为 ansible_templates 表添加风险等级字段
-- ==========================================

-- 添加风险等级字段
ALTER TABLE ansible_templates ADD COLUMN IF NOT EXISTS risk_level VARCHAR(20) DEFAULT 'low';

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_ansible_templates_risk_level ON ansible_templates(risk_level);

-- 添加注释
COMMENT ON COLUMN ansible_templates.risk_level IS '风险等级(low/medium/high)';

-- ==========================================
-- 4. 更新现有记录的默认值
-- ==========================================

-- 将现有清单的环境设置为 dev
UPDATE ansible_inventories SET environment = 'dev' WHERE environment IS NULL OR environment = '';

-- 将现有模板的风险等级设置为 low
UPDATE ansible_templates SET risk_level = 'low' WHERE risk_level IS NULL OR risk_level = '';

