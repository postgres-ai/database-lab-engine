/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { DefaultTheme, makeStyles } from '@material-ui/styles'
import clsx from 'clsx'

import { colors } from '@postgres.ai/shared/styles/colors'

type Props = {
  level: 1 | 2
  tag: 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6'
  text: React.ReactNode
  className?: string
  rightContent?: React.ReactNode
  children?: React.ReactNode
  contentClassName?: string
}

const LEVEL_TO_FONT_SIZE = {
  1: '16px',
  2: '14px',
}

const LEVEL_TO_BOTTOM_PADDING = {
  1: '16px',
  2: '8px',
}

const useStyles = makeStyles<DefaultTheme, { level: Props['level'] }>(
  {
    root: {
      borderBottom: `1px solid ${colors.consoleStroke}`,
    },
    content: (props) => ({
      display: 'flex',
      alignItems: 'center',
      paddingBottom: LEVEL_TO_BOTTOM_PADDING[props.level],
    }),
    text: (props) => ({
      margin: '0 10px 0 0',
      fontSize: LEVEL_TO_FONT_SIZE[props.level],
    }),
  },
  { index: 1 },
)

export const SectionTitle = (props: Props) => {
  const {
    text,
    tag: Tag,
    level,
    rightContent,
    children,
    className,
    contentClassName,
  } = props

  const classes = useStyles({ level })

  return (
    <div className={clsx(classes.root, className)}>
      <div className={clsx(classes.content, contentClassName)}>
        <Tag className={classes.text}>{text}</Tag>
        {rightContent}
      </div>
      {children}
    </div>
  )
}
