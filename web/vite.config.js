import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

// The Go binary embeds web/dist and serves it. In dev, Vite proxies /api to the
// Go process so the frontend and backend feel like one origin.
const DEMO = !!process.env.VITE_DEMO

export default defineConfig(({ command }) => ({
  plugins: [svelte(), tailwindcss()],
  // A real boolean literal (not import.meta.env.VITE_DEMO, which doesn't fold) so
  // prod builds dead-code-eliminate the demo mock + seed entirely.
  define: { __ORIEL_DEMO__: JSON.stringify(DEMO) },
  // The demo is a static GitHub Pages site under /oriel/, so its base is fixed.
  // Otherwise: placeholder base in prod builds that the Go server swaps for
  // ORIEL_BASE_PATH at runtime (one build serves root or a subpath); dev at root.
  base: DEMO ? '/oriel/' : command === 'build' ? '/__ORIEL_BASE__/' : '/',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      // Demo modules do top-level init (const db = …makeContainers()) — a side
      // effect that pins them in prod. Mark side-effect-free so Rollup drops them.
      treeshake: { moduleSideEffects: (id) => !id.includes('/lib/demo/') },
    },
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
