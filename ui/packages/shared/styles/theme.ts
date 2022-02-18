/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { createTheme } from '@material-ui/core'

import { colors } from './colors'

export const theme = createTheme({
  // @ts-ignore
  fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
  palette: {
    primary: {
      main: colors.secondary2.main,
      contrastText: colors.secondary2.contrastText,
      dark: colors.secondary2.dark,
    },
    secondary: {
      main: colors.secondary2.main,
      contrastText: colors.secondary2.contrastText,
      dark: colors.secondary2.dark,
    },
  },
  typography: {
    htmlFontSize: 14,
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    // @ts-ignore
    fontSize: '14px!important',
    h1: {
      fontSize: '16px!important',
    },
    h2: {
      fontSize: '14px!important',
      fontWeight: 'bold',
    },
    h3: {
      fontSize: '14px!important',
      fontWeight: 'normal',
    },
    button: {
      fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
      textTransform: 'unset',
      fontStyle: 'normal',
      fontSize: '14px',
      lineHeight: '18px',
      alignItems: 'center',
      textAlign: 'center',
    },
  },
})
