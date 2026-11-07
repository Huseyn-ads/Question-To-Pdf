/// <reference types="vitest/config" />
import path from 'node:path'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The browser always talks to the frontend origin (http://localhost:5173).
// Every /v1/... path — JSON API, the backend-rendered email verification page
// and the Google OAuth start/callback — is forwarded to the Go backend with
// the path untouched. Host is preserved (changeOrigin: false) so the backend
// sees the same origin the browser uses; set PUBLIC_ORIGIN to that origin.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiTarget = env.API_PROXY_TARGET || 'http://localhost:8080'

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: { '@': path.resolve(import.meta.dirname, 'src') },
    },
    server: {
      port: 5173,
      strictPort: true,
      proxy: {
        '/v1': { target: apiTarget, changeOrigin: false },
        '/health': { target: apiTarget, changeOrigin: false },
        '/ready': { target: apiTarget, changeOrigin: false },
      },
    },
    test: {
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      css: false,
    },
  }
})
