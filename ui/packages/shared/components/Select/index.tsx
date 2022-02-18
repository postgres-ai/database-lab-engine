/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { MenuItem } from '@material-ui/core'

import { TextField, TextFieldProps } from '@postgres.ai/shared/components/TextField'

type Value = string | number

type Props = {
  items: {
    value: Value
    children: React.ReactNode
  }[]
  value: Value | null
  onChange: TextFieldProps['onChange']
  className?: TextFieldProps['className']
  label: TextFieldProps['label']
  fullWidth?: TextFieldProps['fullWidth']
  disabled?: TextFieldProps['disabled']
  error?: boolean
}

export const Select = (props: Props) => {
  const { items, ...textFieldProps } = props
  return (
    <TextField select {...textFieldProps}>
      {items.map((item, i) => {
        return (
          <MenuItem value={item.value} key={`${item.value}-${i}`}>
            {item.children}
          </MenuItem>
        )
      })}
    </TextField>
  )
}
