import { ref, onMounted, onUnmounted } from 'vue'

/**
 * 页面可见性检测 Composable
 * 用于优化轮询机制，当页面不可见时暂停轮询
 */
export function usePageVisibility() {
  // 初始化为当前页面可见性状态
  const isVisible = ref(!document.hidden)

  // 处理可见性变化
  const handleVisibilityChange = () => {
    isVisible.value = !document.hidden
  }

  // 组件挂载时添加事件监听
  onMounted(() => {
    document.addEventListener('visibilitychange', handleVisibilityChange)
  })

  // 组件卸载时移除事件监听
  onUnmounted(() => {
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  })

  return {
    isVisible
  }
}

