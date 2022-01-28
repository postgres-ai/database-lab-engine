import React from 'react'
import ReactDOM from 'react-dom'
import { ThemeProvider } from '@material-ui/core'

import { theme } from '@postgres.ai/shared/styles/theme'

import './index.scss'

import { App } from './App'

ReactDOM.render(
  <React.StrictMode>
    <ThemeProvider theme={theme}>
      <App />
    </ThemeProvider>
  </React.StrictMode>,
  document.getElementById('root'),
)
