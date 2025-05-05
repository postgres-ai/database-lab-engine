/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { colors } from './colors'
import { theme } from './theme'

export const styles: Record<string, any> = {
  root: {
    'minHeight': '100%',
    width: '100%',
    'zIndex': 1,
    position: 'relative',
    [theme.breakpoints.down('sm')]: {
      maxWidth: '100vw',
    },
    [theme.breakpoints.up('md')]: {
      maxWidth: 'calc(100vw - 200px)',
    },
    [theme.breakpoints.up('lg')]: {
      maxWidth: 'calc(100vw - 200px)',
    },
    '& h2': {
      ...theme.typography.h2,
    },
    '& h3': {
      ...theme.typography.h3,
    },
    '& h4': {
      ...theme.typography.h4,
    },
    '& th': {
      fontSize: '14px',
      lineHeight: '16px',
      fontWeight: 'bold',
      color: colors.consoleFadedFont,
    },
  },
  inputField: {
    'margin-bottom': '10px',
    '& > div.MuiFormControl- > label': {
      fontSize: '14px!important',
    },
    '& .MuiOutlinedInput-input, & .MuiOutlinedInput-multiline, & .MuiSelect-select':
      {
        padding: '8px!important',
        fontSize: 14,
      },
    '& .MuiSelect-icon': {
      fontSize: 22,
    },
    '& .MuiInputBase-multiline': {
      padding: '0px!important',
    },
  },
  inputFieldLabel: {
    fontSize: 14,
  },
  inputFieldHelper: {
    fontSize: 11,
    marginLeft: '10px',
  },
  checkbox: {
    'font-size': '14px!important',
    '& > span.MuiFormControlLabel-label': {
      fontSize: 14,
    },
  },

  tableHead: {
    height: '30px',
    lineHeight: '30px',
    position: 'relative',
  },
  tableHeadActions: {
    position: 'absolute',
    right: '0px',
    top: '0px',
  },
  bottomSpace: {
    display: 'block',
    height: 130,
  },
}
