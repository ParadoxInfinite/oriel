import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

// The Go binary embeds web/dist and serves it. In dev, Vite proxies /api to the
// Go process so the frontend and backend feel like one origin.
export default defineConfig(({ command }) => ({
  plugins: [svelte(), tailwindcss()],
  // Placeholder base in prod builds; the Go server swaps it for ORIEL_BASE_PATH
  // at runtime so one build serves at root or a subpath. Dev stays at root.
  base: command === 'build' ? '/__ORIEL_BASE__/' : '/',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:4321',
        changeOrigin: true,
      },
    },
  },
}))
