import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

// The Go binary embeds web/dist and serves it. In dev, Vite proxies /api to the
// Go process so the frontend and backend feel like one origin.
const DEMO = !!process.env.VITE_DEMO

export default defineConfig(({ command }) => ({
  plugins: [svelte(), tailwindcss()],
  // The demo is a static GitHub Pages site under /oriel/, so its base is fixed.
  // Otherwise: placeholder base in prod builds that the Go server swaps for
  // ORIEL_BASE_PATH at runtime (one build serves root or a subpath); dev at root.
  base: DEMO ? '/oriel/' : command === 'build' ? '/__ORIEL_BASE__/' : '/',
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
