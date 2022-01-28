/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/core'

import { Status, Props as StatusProps } from '@postgres.ai/shared/components/Status'

type Props = {
  children: React.ReactNode
  type?: StatusProps['type']
  icon?: StatusProps['icon']
}

const useStyles = makeStyles({
  root: {
    fontWeight: 400,
    marginTop: '2px',
    fontSize: '10px',
  },
  icon: {
    alignSelf: 'flex-start',
  },
})

export const ValueStatus = (props: Props) => {
  const { children, type, icon } = props

  const classes = useStyles()

  return (
    <Status
      className={classes.root}
      classNameIcon={classes.icon}
      type={type}
      icon={icon}
    >
      {children}
    </Status>
  )
}
