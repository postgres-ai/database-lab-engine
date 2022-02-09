/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Status, Props as StatusProps } from '@postgres.ai/shared/components/Status'

import styles from './styles.module.scss'

type Props = {
  messages: {
    type: StatusProps['type']
    value: string
  }[]
}

export const Banner = (props: Props) => {
  const { messages } = props

  return (
    <div className={styles.root}>
      {messages.map((message) => {
        return (
          <Status
            key={`${message.type}-${message.value}`}
            className={styles.content}
            type={message.type}
          >
            {message.value}
          </Status>
        )
      })}
    </div>
  )
}
