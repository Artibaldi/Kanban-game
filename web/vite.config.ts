import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  // The 3D engine (three.js) is one lazy chunk of ~550 kB loaded only on game screens.
  build: { chunkSizeWarningLimit: 650 },
  server: {
    port: 5173,
    strictPort: true,
    proxy: { '/api': { target: process.env.KG_API_TARGET || 'http://127.0.0.1:8090', changeOrigin: false } },
  },
})
