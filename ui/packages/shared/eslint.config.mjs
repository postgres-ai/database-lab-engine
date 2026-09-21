import globals from 'globals'

import { baseConfig } from '../../eslint.config.base.mjs'

export default [
  ...baseConfig,
  {
    // Build scripts run under Node, outside the bundler, and stay CommonJS.
    files: ['scripts/**/*.js'],
    languageOptions: {
      sourceType: 'commonjs',
      globals: globals.node,
    },
    rules: {
      '@typescript-eslint/no-require-imports': 'off',
    },
  },
  {
    ignores: ['build-tmp/'],
  },
]
