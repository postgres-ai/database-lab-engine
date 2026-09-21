import cypressPlugin from 'eslint-plugin-cypress'

import { baseConfig } from '../../eslint.config.base.mjs'

export default [
  ...baseConfig,
  {
    files: ['cypress/**/*.{ts,tsx}'],
    plugins: {
      cypress: cypressPlugin,
    },
    rules: {
      ...cypressPlugin.configs.recommended.rules,
    },
  },
]
