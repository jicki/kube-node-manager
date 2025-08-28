/// <reference types="vite/client" />

// 声明构建时注入的全局变量
declare const __APP_VERSION__: string
declare const __BUILD_TIME__: string

// 声明环境变量类型
interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly VITE_API_TARGET: string
  readonly VITE_ENABLE_LDAP: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
