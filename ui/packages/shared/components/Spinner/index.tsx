import cn from 'classnames'
import { SpinnerIcon } from './icon'

import styles from './styles.module.scss'

export type Size = 'sm' | 'md' | 'lg'

export type Props = {
  size?: Size
  className?: string
}

export const Spinner = (props: Props) => {
  const { size = 'md' } = props
  return <SpinnerIcon className={cn(styles.root, styles[size], props.className)} />
}
