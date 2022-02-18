/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import clsx from 'clsx'
import copy from 'copy-to-clipboard'
import useCountDown from 'react-countdown-hook'

import { Button } from '@postgres.ai/shared/components/Button2'

import styles from './styles.module.scss'

type Props = {
  value: string
  className?: string
}

const COOLDOWN_INTERVAL = 1000
const COOLDOWN_TIME = 1000

export const FormattedText = (props: Props) => {
  const { value, className } = props

  const [timeLeft, { start }] = useCountDown(0, COOLDOWN_INTERVAL)

  const handleCopy = () => {
    copy(value)
    start(COOLDOWN_TIME)
  }

  return (
    <div className={clsx(styles.root, className)}>
      <pre className={styles.content}>{value}</pre>
      <div className={styles.copyButtonContainer}>
        <Button size="sm" onClick={handleCopy} className={styles.copyButton}>
          {timeLeft ? 'Copied' : 'Copy'}
        </Button>
      </div>
    </div>
  )
}
