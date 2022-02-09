import React from 'react'
import cn from 'classnames'

import { Spinner } from '@postgres.ai/shared/components/Spinner'

import styles from './styles.module.scss'

type Props = {
  type?: React.ButtonHTMLAttributes<HTMLButtonElement>['type']
  children: React.ReactNode
  onClick?: React.DOMAttributes<HTMLButtonElement>['onClick']
  size?: 'sm' | 'md' | 'lg'
  theme?: 'primary' | 'secondary' | 'accent'
  className?: string
  isDisabled?: boolean
  isLoading?: boolean
}

export const Button = React.forwardRef<HTMLButtonElement, Props>(
  (props, ref) => {
    const { type = 'button', size = 'md', theme = 'secondary' } = props

    const isDisabled = props.isDisabled || props.isLoading

    return (
      <button
        ref={ref}
        className={cn(
          styles.root,
          styles[size],
          styles[theme],
          props.className,
        )}
        type={type}
        onClick={props.onClick}
        disabled={isDisabled}
      >
        {props.children}
        {props.isLoading && <Spinner size="sm" className={styles.spinner} />}
      </button>
    )
  },
)
