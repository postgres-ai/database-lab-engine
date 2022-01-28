import cn from 'classnames'

import styles from './styles.module.scss'

type Props = {
  children: React.ReactNode
  className?: string
}

export const PageContainer = (props: Props) => {
  return (
    <div className={cn(styles.root, props.className)}>{props.children}</div>
  )
}
