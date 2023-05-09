/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import cn from 'classnames'
import { useEffect, useState } from 'react'
import { makeStyles } from '@material-ui/core'

import { Button } from '@postgres.ai/shared/components/MenuButton'
import { useWindowDimensions } from '@postgres.ai/shared/hooks/useWindowDimensions'
import { ReactComponent as ArrowRightIcon } from '@postgres.ai/ce/src/App/Menu/icons/arrow-right.svg'
import { ReactComponent as ArrowLeftIcon } from '@postgres.ai/ce/src/App/Menu/icons/arrow-left.svg'

import { Status } from './Status'
import { Retrieval } from './Retrieval'
import { Connection } from './Connection'
import { Disks } from './Disks'
import { Snapshots } from './Snapshots'

const SIDEBAR_COLLAPSED_PARAM = 'overviewSidebarCollapsed'
const SMALL_BREAKPOINT_PX = 937

const useStyles = makeStyles(
  (theme) => ({
    container: {
      minHeight: 0,
      minWidth: 0,
      width: '437px',
      transition: 'width 0.2s ease-out',

      [theme.breakpoints.down('sm')]: {
        width: '100%',
      },
    },
    root: {
      flex: '0 0 auto',

      [theme.breakpoints.down('md')]: {
        width: '300px',
      },

      [theme.breakpoints.down('sm')]: {
        flex: '0 0 auto',
        width: '100%',
        marginTop: '20px',
      },
    },
    collapsed: {
      height: '100%',
      width: '64px',
    },
    collapseBtn: {
      background: '#f9f9f9',
      margin: '20px 0 10px 0',
      fontWeight: 500,

      '& svg': {
        marginRight: '5px',
      },

      '&:hover': {
        background: '#f1f1f1',
      },
    },
    arrowImage: {
      width: '16px',
      height: '16px',

      '& path': {
        fill: '#000',
      },
    },
  }),
  { index: 1 },
)

export const Info = () => {
  const classes = useStyles()
  const width = useWindowDimensions()
  const [onHover, setOnHover] = useState(false)
  const isMobileScreen = width <= SMALL_BREAKPOINT_PX

  const [isCollapsed, setIsCollapsed] = useState(
    () => localStorage.getItem(SIDEBAR_COLLAPSED_PARAM) === '1',
  )

  const handleClick = () => {
    setIsCollapsed(!isCollapsed)
    localStorage.setItem(SIDEBAR_COLLAPSED_PARAM, isCollapsed ? '0' : '1')
  }

  useEffect(() => {
    if (isMobileScreen) {
      setIsCollapsed(false)
    }
  }, [width])

  return (
    <div
      className={cn(
        classes.container,
        !isCollapsed ? classes.root : classes.collapsed,
      )}
    >
      {!isMobileScreen && (
        <Button
          onMouseEnter={() => setOnHover(true)}
          onMouseLeave={() => setOnHover(false)}
          className={classes.collapseBtn}
          onClick={handleClick}
          isCollapsed={isCollapsed}
          type="button"
          icon={
            isCollapsed ? (
              <ArrowLeftIcon className={classes.arrowImage} />
            ) : (
              <ArrowRightIcon className={classes.arrowImage} />
            )
          }
        >
          {onHover && 'Collapse'}
        </Button>
      )}

      {!isCollapsed && (
        <div>
          <Status />
          <Retrieval />
          <Connection />
          <Disks />
          <Snapshots />
        </div>
      )}
    </div>
  )
}
