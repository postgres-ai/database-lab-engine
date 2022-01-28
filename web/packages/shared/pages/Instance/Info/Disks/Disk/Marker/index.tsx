import React from 'react'
import { makeStyles } from '@material-ui/core'
import clsx from 'clsx'

import { CircleIcon } from '@postgres.ai/shared/icons/Circle'

type Props = {
  className: string
}

const useStyles = makeStyles({
  root: {
    display: 'inline',
    verticalAlign: 'middle',
    width: '10px',
  }
})

export const Marker = (props: Props) => {
  const classes = useStyles()

  return <CircleIcon className={clsx(classes.root, props.className)} />
}
