import { mkdirSync, writeFileSync } from 'fs'
import path from 'path'
import { pathToFileURL } from 'node:url'
/// <reference types="vitest" />
import { defineConfig, type Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import tsconfigPaths from 'vite-tsconfig-paths'
import checker from 'vite-plugin-checker'

const buildTimestamp = Date.now()

/**
 * Writes build/meta.json after bundling so the running app can expose
 * build metadata (timestamp, date) without embedding it in JS chunks.
 */
function metaJsonPlugin(): Plugin {
  return {
    name: 'meta-json',
    apply: 'build',
    closeBundle() {
      const outDir = path.resolve(__dirname, 'build')
      const meta = {
        buildTimestamp,
        buildDate: buildTimestamp, // backward compat with older UI versions
      }
      mkdirSync(outDir, { recursive: true })
      writeFileSync(path.resolve(outDir, 'meta.json'), JSON.stringify(meta))
    },
  }
}

// custom importer needed because Vite's resolve.alias does not apply to CSS/SCSS @use/@import paths
const sharedScssImporter = {
  findFileUrl(url: string) {
    if (url.startsWith('@postgres.ai/shared/')) {
      const resolved = url.replace(
        '@postgres.ai/shared/',
        path.resolve(__dirname, '../shared') + '/',
      )
      return pathToFileURL(resolved)
    }
    return null
  },
}

const devProxyTarget =
  process.env.VITE_DEV_PROXY_TARGET || 'http://localhost:446'

export default defineConfig({
  plugins: [
    react(),
    tsconfigPaths(),
    checker({
      typescript: true,
    }),
    metaJsonPlugin(),
  ],
  define: {
    // BUILD_TIMESTAMP injected at build time; uses process.env shim for legacy env.ts
    'process.env.BUILD_TIMESTAMP': JSON.stringify(String(buildTimestamp)),
  },
  server: {
    port: 3001,
    proxy: {
      '/api': {
        target: devProxyTarget,
        changeOrigin: true,
      },
      '/ws': {
        target: devProxyTarget,
        changeOrigin: true,
        ws: true,
      },
    },
  },
  resolve: {},
  css: {
    preprocessorOptions: {
      scss: {
        importers: [sharedScssImporter],
      },
    },
  },
  build: {
    outDir: 'build',
    sourcemap: 'hidden',
  },
  test: {
    environment: 'happy-dom',
    globals: false,
    setupFiles: ['./src/test/setup.ts'],
  },
})
