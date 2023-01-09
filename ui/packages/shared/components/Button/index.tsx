/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { forwardRef } from 'react'
import { makeStyles } from '@material-ui/styles'
import { Button as ButtonBase, ButtonProps } from '@material-ui/core'
import clsx from 'clsx'

import { colors } from '@postgres.ai/shared/styles/colors'

type Props = Omit<
  ButtonProps,
  'variant' | 'color' | 'className' | 'disabled'
> & {
  variant?: 'primary' | 'secondary'
  className?: string
  isDisabled?: boolean
}

const useStyles = makeStyles(
  {
    root: {
      whiteSpace: 'nowrap',

      '&.MuiButton-outlinedPrimary': {
        background: colors.white,

        '&:hover': {
          background: '#f4f6f7',
        },
      },

      '&:disabled': {
        cursor: 'not-allowed',
        pointerEvents: 'all',

        '&.MuiButton-outlinedPrimary:hover': {
          background: colors.white,
          border: '1px solid rgba(0, 0, 0, 0.12)',
        },
      },
    },
  },
  { index: 1 },
)

const VARIANT_MAP = {
  primary: 'contained' as const,
  secondary: 'outlined' as const,
}

export const Button = forwardRef(
  (props: Props, ref: React.Ref<HTMLButtonElement>) => {
    const {
      variant = 'secondary',
      className,
      isDisabled,
      size = 'small',
      ...buttonProps
    } = props
    const classes = useStyles()

    return (
      <ButtonBase
        {...buttonProps}
        size={size}
        ref={ref}
        disabled={isDisabled}
        className={clsx(classes.root, className)}
        variant={VARIANT_MAP[variant]}
        color="primary"
      />
    )
  },
)
