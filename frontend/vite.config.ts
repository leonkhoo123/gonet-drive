import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'
import { execSync } from 'child_process'

// Generate version info from Git + build date
let commitHash = 'dev'
try {
  commitHash = execSync('git rev-parse --short HEAD').toString().trim()
} catch {
  console.warn('⚠️ Git hash not found — using "dev" version tag')
}
// const buildDate = new Date().toISOString().split('T')[0]
const now = new Date()
const gmt8Time = new Date(now.getTime() + 8 * 60 * 60 * 1000) // add 8 hours
const buildTime = gmt8Time.toISOString().replace('T', ' ').split('.')[0] // "YYYY-MM-DD HH:MM:SS"

const appVersion = `${buildTime}-${commitHash}`

// Vite configuration
export default defineConfig({
  define: {
    __APP_VERSION__: JSON.stringify(appVersion), // available globally in app
  },
  plugins: [
    react({
      babel: {
        plugins: [['babel-plugin-react-compiler']],
      },
    }),
    tailwindcss(),
    VitePWA({
      registerType: 'prompt',
      devOptions: {
        enabled: true,
      },
      includeAssets: ['favicon.ico', 'apple-touch-icon.png', 'robots.txt'],
      manifest: {
        name: 'Demo Cloud',
        short_name: 'Demo Cloud',
        start_url: '/',
        display: 'standalone',
        background_color: '#ffffff',
        theme_color: '#ffffff',
        icons: [
          {
            src: '/images/logo-removebg-preview.png',
            sizes: '192x192',
            type: 'image/png',
          },
          {
            src: '/images/logo-removebg-preview.png',
            sizes: '512x512',
            type: 'image/png',
          },
        ],
      },
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
