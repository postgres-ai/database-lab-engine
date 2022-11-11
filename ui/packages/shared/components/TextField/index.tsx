/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import {
  TextField as TextFieldBase,
  makeStyles,
  InputProps,
  TextFieldProps as TextFieldPropsBase,
} from '@material-ui/core'
import clsx from 'clsx'

export type TextFieldProps = {
  label?: string
  disabled?: boolean
  defaultValue?: string
  value?: TextFieldPropsBase['value']
  className?: string
  fullWidth?: boolean
  autoFocus?: boolean
  id?: TextFieldPropsBase['id']
  multiline?: TextFieldPropsBase['multiline']
  onKeyDown?: TextFieldPropsBase['onKeyDown']
  onChange?: TextFieldPropsBase['onChange']
  InputProps?: InputProps
  InputLabelProps?: TextFieldPropsBase['InputLabelProps']
  children?: TextFieldPropsBase['children']
  select?: TextFieldPropsBase['select']
  type?: 'text' | 'password'
  error?: boolean
  placeholder?: string
}

const useStyles = makeStyles(
  {
    root: {
      fontSize: '14px',
    },
    selectIcon: {
      fontSize: '24px',
    },
    inputRoot: {
      padding: 0,
    },
    input: {
      padding: '8px',
    },
  },
  { index: 1 },
)

export const TextField = (props: TextFieldProps) => {
  const classes = useStyles()

  return (
    <TextFieldBase
      onKeyDown={props.onKeyDown}
      autoFocus={props.autoFocus}
      id={props.id}
      multiline={props.multiline}
      label={props.label}
      variant="outlined"
      disabled={props.disabled}
      className={clsx(classes.root, props.className)}
      defaultValue={props.defaultValue}
      value={props.value}
      margin="normal"
      fullWidth={props.fullWidth}
      classes={{}}
      InputProps={{
        ...props.InputProps,

        classes: {
          root: classes.inputRoot,
          input: classes.input,
          ...props.InputProps?.classes,
        },
      }}
      SelectProps={{
        classes: {
          icon: classes.selectIcon,
        },
      }}
      InputLabelProps={{
        shrink: true,

        ...props.InputLabelProps,
      }}
      onChange={props.onChange}
      children={props.children}
      select={props.select}
      type={props.type}
      error={props.error}
      placeholder={props.placeholder}
    />
  )
}
