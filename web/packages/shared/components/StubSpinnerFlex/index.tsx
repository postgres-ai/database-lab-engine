import cn from 'classnames'

import { Spinner, Size } from '@postgres.ai/shared/components/Spinner'

import styles from './styles.module.scss'

type Props = {
  mode?: 'absolute' | 'flex'
  size?: Size
  className?: string
}

export const StubSpinner = (props: Props) => {
  const { mode = 'flex', size = 'lg' } = props
  return (
    <div className={cn(styles.root, props.className, styles[mode])}>
      <Spinner size={size} />
    </div>
  )
}
