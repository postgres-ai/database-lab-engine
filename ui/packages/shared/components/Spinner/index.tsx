import cn from 'classnames'

import { ReactComponent as Icon } from './icon.svg'

import styles from './styles.module.scss'

export type Size = 'sm' | 'md' | 'lg'

export type Props = {
  size?: Size
  className?: string
}

export const Spinner = (props: Props) => {
  const { size = 'md' } = props
  return <Icon className={cn(styles.root, styles[size], props.className)} />
}
