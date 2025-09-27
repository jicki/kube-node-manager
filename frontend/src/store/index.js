import { createPinia } from 'pinia'

const pinia = createPinia()

// 开发环境下启用 Pinia devtools
if (process.env.NODE_ENV === 'development') {
  pinia.use(({ store }) => {
    // 添加调试信息
    store.$onAction(({ name, args, after, onError }) => {
      const startTime = Date.now()
      console.log(`🚀 开始执行 "${store.$id}.${name}"`, args)

      after((result) => {
        console.log(`✅ 完成 "${store.$id}.${name}" (${Date.now() - startTime}ms)`, result)
      })

      onError((error) => {
        console.error(`❌ 错误 "${store.$id}.${name}" (${Date.now() - startTime}ms)`, error)
      })
    })
  })
}

export default pinia