/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import clsx from 'clsx'

import { WarningIcon } from '@postgres.ai/shared/icons/Warning'
import { CircleIcon } from '@postgres.ai/shared/icons/Circle'

import styles from './styles.module.scss'

type Type = 'ok' | 'warning' | 'error' | 'waiting' | 'unknown'

export type Props = {
  type?: Type
  children?: React.ReactNode
  icon?: React.ReactNode
  className?: string
  classNameIcon?: string
  disableColor?: boolean
}

const TYPE_TO_ICON = {
  ok: <CircleIcon className={styles.icon} />,
  warning: <WarningIcon className={styles.icon} />,
  error: <WarningIcon className={styles.icon} />,
  waiting: <CircleIcon className={styles.icon} />,
  unknown: <CircleIcon className={styles.icon} />,
}

export const Status = React.forwardRef<HTMLDivElement, Props>((props, ref) => {
  const {
    type = 'ok',
    children = type,
    icon = TYPE_TO_ICON[type],
    className,
    classNameIcon,
    disableColor,
    ...hiddenProps
  } = props

  return (
    <span
      {...hiddenProps}
      className={clsx(styles.root, !disableColor && styles[type], className)}
      ref={ref}
    >
      {icon && (
        <span className={clsx(styles.iconContainer, classNameIcon)}>
          {icon}
          &thinsp;
        </span>
      )}
      {children}
    </span>
  )
})
