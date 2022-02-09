import cn from 'classnames'

import styles from './styles.module.scss'

type Props = {
  children: React.ReactNode
  className?: string
  onSubmit?: React.FormEventHandler<HTMLFormElement>
}

export const Card = (props: Props) => {
  return (
    <form
      onSubmit={props.onSubmit}
      className={cn(styles.root, props.className)}
    >
      {props.children}
    </form>
  )
}
