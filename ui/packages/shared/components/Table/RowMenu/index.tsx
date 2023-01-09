/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useState } from 'react'
import { Menu, MenuItem, IconButton, makeStyles } from '@material-ui/core'
import { MoreVert } from '@material-ui/icons'
import clsx from 'clsx'

import { StubSpinner } from '@postgres.ai/shared/components/StubSpinnerFlex'

const DIRECTION_TO_ORIGIN = {
  left: 'right',
  right: 'left',
} as const

type Action = {
  name: string
  onClick: () => void
  isDisabled?: boolean
}

export type Props = {
  actions: Action[]
  isDisabled?: boolean
  isLoading?: boolean
  direction?: 'left' | 'right'
}

const useStyles = makeStyles(
  {
    button: {
      '&:disabled': {
        cursor: 'not-allowed',
        pointerEvents: 'all',
      },
    },
    spinner: {
      background: 'transparent',
    },
    hiddenIcon: {
      visibility: 'hidden',
    },
  },
  { index: 1 },
)

export const RowMenu = (props: Props) => {
  const {
    actions,
    isDisabled = false,
    isLoading = false,
    direction = 'right',
  } = props
  const classes = useStyles()
  const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null)

  const isOpen = Boolean(anchorEl)

  const openMenu: React.MouseEventHandler<HTMLButtonElement> = (e) =>
    setAnchorEl(e.currentTarget)

  const closeMenu = () => setAnchorEl(null)

  return (
    <>
      <IconButton
        onClick={openMenu}
        disabled={isDisabled || isLoading}
        className={classes.button}
      >
        <MoreVert className={clsx(isLoading && classes.hiddenIcon)} />
        {isLoading && (
          <StubSpinner size="sm" mode="absolute" className={classes.spinner} />
        )}
      </IconButton>
      <Menu
        anchorOrigin={{
          vertical: 'top',
          horizontal: DIRECTION_TO_ORIGIN[direction],
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: DIRECTION_TO_ORIGIN[direction],
        }}
        anchorEl={anchorEl}
        keepMounted
        open={isOpen}
        onClose={closeMenu}
      >
        {actions.map((action, i) => {
          return (
            <MenuItem
              key={i}
              onClick={() => {
                closeMenu()
                action.onClick()
              }}
              disabled={action.isDisabled}
            >
              {action.name}
            </MenuItem>
          )
        })}
      </Menu>
    </>
  )
}
