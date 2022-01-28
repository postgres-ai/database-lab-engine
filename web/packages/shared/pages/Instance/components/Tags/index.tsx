/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/core'

import { Tag } from './Tag'

export type TagsProps = {
  data: { name: string; value: string }[]
}

const useStyles = makeStyles({
  root: {
    display: 'flex',
    flexWrap: 'wrap',
    marginRight: '-8px',
  },
})

export const Tags = (props: TagsProps) => {
  const classes = useStyles()

  return (
    <div className={classes.root}>
      {props.data.map((tag) => {
        return (
          <Tag key={tag.name} name={tag.name}>
            {tag.value}
          </Tag>
        )
      })}
    </div>
  )
}
