/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/styles'

import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'

type Props = {
  title: string
  children: React.ReactNode
  rightContent?: React.ReactNode
}

const useStyles = makeStyles({
  root: {
    '& + $root': {
      marginTop: '28px',
    },
  },
  content: {
    marginTop: '8px',
  },
})

export const Section = (props: Props) => {
  const { title, children, rightContent } = props

  const classes = useStyles()

  return (
    <div className={classes.root}>
      <SectionTitle
        text={title}
        level={2}
        tag="h3"
        rightContent={rightContent}
      />

      <div className={classes.content}>{children}</div>
    </div>
  )
}
