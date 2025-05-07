/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import cn from 'classnames'
import { useState } from 'react'
import { makeStyles } from '@material-ui/core'

import { Button } from '@postgres.ai/shared/components/MenuButton'
import { useWindowDimensions } from '@postgres.ai/shared/hooks/useWindowDimensions'
import { ArrowLeft, ArrowRight } from '@postgres.ai/shared/pages/Instance/Info/Icons'

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
      background: '#f9f9f9 !important',
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

type InfoProps = {
  hideBranchingFeatures?: boolean
}

export const Info = (props: InfoProps) => {
  const classes = useStyles()
  const width = useWindowDimensions()
  const [onHover, setOnHover] = useState(false)
  const isMobileScreen = width <= SMALL_BREAKPOINT_PX

  const [isCollapsed, setIsCollapsed] = useState(
    () =>
      localStorage.getItem(SIDEBAR_COLLAPSED_PARAM) === '1' && !isMobileScreen,
  )

  const handleClick = () => {
    setIsCollapsed(!isCollapsed)
    localStorage.setItem(SIDEBAR_COLLAPSED_PARAM, isCollapsed ? '0' : '1')
  }

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
              <ArrowLeft className={classes.arrowImage} />
            ) : (
              <ArrowRight className={classes.arrowImage} />
            )
          }
        >
          {onHover && 'Collapse'}
        </Button>
      )}

      {!isCollapsed && (
        <div>
          <Status />
          <Retrieval hideBranchingFeatures={props.hideBranchingFeatures} />
          <Connection />
          <Disks />
          <Snapshots />
        </div>
      )}
    </div>
  )
}
