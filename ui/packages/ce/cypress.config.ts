import { defineConfig } from 'cypress'

export default defineConfig({
  pageLoadTimeout: 10000,
  defaultCommandTimeout: 10000,
  e2e: {
    testIsolation: false,
    supportFile: false,
    baseUrl: 'http://localhost:3001',
    screenshotOnRunFailure: false,
    video: false,
  },

  component: {
    devServer: {
      framework: 'create-react-app',
      bundler: 'webpack',
    },
  },
})
