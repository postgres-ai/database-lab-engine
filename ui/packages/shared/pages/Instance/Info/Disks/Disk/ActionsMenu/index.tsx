/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useState } from 'react'
import { IconButton, makeStyles, Menu, MenuItem } from '@material-ui/core'
import { MoreVert } from '@material-ui/icons'
import copy from 'copy-to-clipboard'

import { colors } from '@postgres.ai/shared/styles/colors'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'

type Props = {
  poolId: string | null
  poolName: string
  isActive: boolean
}

const useStyles = makeStyles(
  {
    root: {
      padding: '3px 1px',
      border: `1px solid ${colors.consoleStroke}`,
      borderRadius: '4px',
    },
  },
  { index: 1 },
)

export const ActionsMenu = (props: Props) => {
  const classes = useStyles()
  const stores = useStores()

  const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null)

  const isOpen = Boolean(anchorEl)

  const openMenu = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation()
    setAnchorEl(e.currentTarget)
  }

  const closeMenu = () => setAnchorEl(null)

  return (
    <>
      <IconButton className={classes.root} onClick={openMenu}>
        <MoreVert />
      </IconButton>

      <Menu
        anchorOrigin={{
          vertical: 'top',
          horizontal: 'left',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'left',
        }}
        anchorEl={anchorEl}
        keepMounted
        open={isOpen}
        onClose={closeMenu}
        onClick={(e) => e.stopPropagation()}
      >
        <MenuItem
          onClick={() => {
            closeMenu()
            copy(props.poolName)
          }}
        >
          Copy name
        </MenuItem>
        {props.isActive && (
          <>
            <MenuItem
              onClick={() => {
                closeMenu()
                stores.clonesModal.openModal({ pool: props.poolId })
              }}
            >
              List clones
            </MenuItem>
            <MenuItem
              onClick={() => {
                closeMenu()
                stores.snapshotsModal.openModal({ pool: props.poolId })
              }}
            >
              List snapshots
            </MenuItem>
          </>
        )}
      </Menu>
    </>
  )
}
