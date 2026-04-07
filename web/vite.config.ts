import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  base: '/paper_mind/',
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/paper_mind_api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/paper_mind_api/, '/api'),
      },
      '/healthz': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})