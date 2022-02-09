/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/core'

import { Button } from '@postgres.ai/shared/components/Button'

type Props = {
  items: {
    text: string
    variant?: 'primary' | 'secondary'
    onClick: () => void
    isDisabled?: boolean
  }[]
}

const useStyles = makeStyles({
  root: {
    display: 'flex',
    justifyContent: 'flex-end',
    marginTop: '16px',
  },
  button: {
    marginLeft: '8px',
  },
})

export const SimpleModalControls = (props: Props) => {
  const classes = useStyles()

  return (
    <div className={classes.root}>
      {props.items.map((item) => {
        return (
          <Button
            key={item.text}
            className={classes.button}
            onClick={item.onClick}
            variant={item.variant}
            isDisabled={item.isDisabled}
          >
            {item.text}
          </Button>
        )
      })}
    </div>
  )
}
