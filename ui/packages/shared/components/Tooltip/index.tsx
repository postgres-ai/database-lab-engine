/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {
  Tooltip as TooltipBase,
  TooltipProps,
  makeStyles,
} from '@material-ui/core'

type Props = {
  children: JSX.Element
  content: string | JSX.Element
  disableTouchListener?: boolean
  placement?: TooltipProps['placement']
  interactive?: TooltipProps['interactive']
  enterTouchDelay?: TooltipProps['enterTouchDelay']
}

const useStyles = makeStyles(
  {
    tooltip: {
      fontSize: '10px',
      padding: '4px 8px',
    },
  },
  { index: 1 },
)

export const Tooltip = (props: Props) => {
  const {
    content,
    placement = 'top',
    enterTouchDelay = 0,
    ...otherProps
  } = props

  const classes = useStyles()

  return (
    <TooltipBase
      {...otherProps}
      enterTouchDelay={enterTouchDelay}
      placement={placement}
      title={content}
      classes={{ tooltip: classes.tooltip }}
    />
  )
}
