const path = require('path')

const {
  getLoader,
  loaderByName,
  getPlugin,
  pluginByName,
  addPlugins,
  whenProd,
} = require('@craco/craco')

const CreateFilePlugin = require('create-file-webpack')

const SHARED_PACKAGE_PATH = path.join(__dirname, '../shared')

const buildTimestamp = Date.now()

module.exports = (context) => ({
  webpack: {
    configure: (webpackConfig) => {
      // Add shared to resolve.
      const forkTsCheckerWebpackPlugin = getPlugin(
        webpackConfig,
        pluginByName('ForkTsCheckerWebpackPlugin'),
      )

      if (forkTsCheckerWebpackPlugin.isFound) {
        forkTsCheckerWebpackPlugin.match.reportFiles = []
        forkTsCheckerWebpackPlugin.match.options.reportFiles = []
      }

      const babelLoader = getLoader(webpackConfig, loaderByName('babel-loader'))

      if (babelLoader.isFound) {
        if (Array.isArray(babelLoader.match.loader.include)) {
          babelLoader.match.loader.include.push(SHARED_PACKAGE_PATH)
        } else {
          babelLoader.match.loader.include = [
            babelLoader.match.loader.include,
            SHARED_PACKAGE_PATH,
          ]
        }
      }

      // Generating meta.json
      whenProd(() =>
        addPlugins(webpackConfig, [
          new CreateFilePlugin({
            path: context.paths.appBuild,
            fileName: './meta.json',
            content: JSON.stringify({
              buildTimestamp,
              // Backward compatibility with prev versions UI.
              buildDate: buildTimestamp,
            }),
          }),
        ]),
      )

      // Passing build timestamp.
      const { match: definePlugin } = getPlugin(webpackConfig, pluginByName('DefinePlugin'))
      if (definePlugin) definePlugin.definitions['process.env'].BUILD_TIMESTAMP = `${buildTimestamp}`

      return webpackConfig
    },
  },
})
