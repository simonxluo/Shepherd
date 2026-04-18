import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { i18nReady } from './lib/i18n'
import { configLoader } from './lib/config'
import { updateApiClientUrl } from './lib/api/client'

// Initialize app
async function initApp() {
  // Wait for i18n to initialize
  await i18nReady

  try {
    // Load frontend config
    const config = await configLoader.load()

    // Update API client backend URL
    updateApiClientUrl(config.api.baseUrl + config.api.basePath)

    console.log('Frontend config loaded:', {
      backendUrl: config.api.baseUrl,
      features: config.features,
      ui: config.ui,
    })
  } catch (error) {
    console.error('Failed to load frontend config:', error)
    // Continue with default config
  }

  // Render app
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}

// Start app
initApp()
