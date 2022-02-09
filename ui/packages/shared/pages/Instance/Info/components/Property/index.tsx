/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import cn from 'classnames'

import styles from './styles.module.scss'

type Props = {
  name: React.ReactNode
  children: React.ReactNode
  classes?: {
    name?: string
    content?: string
  }
}

export const Property = (props: Props) => {
  const { name, children } = props

  return (
    <div className={styles.root}>
      <label className={cn(styles.name, props.classes?.name)}>{name}</label>
      <div className={cn(styles.content, props.classes?.content)}>
        {children}
      </div>
    </div>
  )
}
