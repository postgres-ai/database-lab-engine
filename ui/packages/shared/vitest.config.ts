import { defineConfig } from 'vitest/config'
import tsconfigPaths from 'vite-tsconfig-paths'

export default defineConfig({
  // Sources import each other through the published package name, which only resolves once
  // the package is installed; tsconfig `paths` maps it back to this directory.
  plugins: [tsconfigPaths()],
  test: {
    environment: 'happy-dom',
    globals: false,
    setupFiles: ['./test/setup.ts'],
    include: ['**/*.test.{ts,tsx}'],
    exclude: ['node_modules/**', 'dist/**', 'build-tmp/**'],
  },
})
