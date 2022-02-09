/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/core'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { colors } from '@postgres.ai/shared/styles/vars'

type Props = {
  name: string
  children: React.ReactNode
}

const useStyles = makeStyles({
  root: {
    fontSize: '12px',
    marginRight: '8px',
    marginBottom: '8px',
    borderRadius: '12px',
    background: colors.gray,
  },
  content: {
    padding: '5px 8px',
    whiteSpace: 'nowrap',
  },
  curtainLeft: {
    background: 'linear-gradient(to right, rgba(0,0,0,0.1), transparent);',
  },
  curtainRight: {
    background: 'linear-gradient(to left, rgba(0,0,0,0.1), transparent);',
  },
  value: {
    fontWeight: 700,
  },
})

export const Tag = (props: Props) => {
  const classes = useStyles()

  return (
    <HorizontalScrollContainer
      classes={{
        root: classes.root,
        content: classes.content,
        curtainLeft: classes.curtainLeft,
        curtainRight: classes.curtainRight,
      }}
    >
      {props.name}: <span className={classes.value}>{props.children}</span>
    </HorizontalScrollContainer>
  )
}
