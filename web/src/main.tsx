import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { i18nReady } from './lib/i18n'
import { configLoader } from './lib/config'
import { updateApiClientUrl } from './lib/api/client'

// 初始化应用
async function initApp() {
  // 等待 i18n 初始化完成
  await i18nReady

  try {
    // 加载前端配置
    const config = await configLoader.load()

    // 更新 API 客户端的后端 URL
    updateApiClientUrl(config.api.baseUrl + config.api.basePath)

    console.log('Frontend config loaded:', {
      backendUrl: config.api.baseUrl,
      features: config.features,
      ui: config.ui,
    })
  } catch (error) {
    console.error('Failed to load frontend config:', error)
    // 使用默认配置继续启动
  }

  // 渲染应用
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}

// 启动应用
initApp()
