/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Spinner } from '@postgres.ai/shared/components/Spinner'

import styles from './styles.module.scss'

export const PageSpinner = () => {
  return (
    <div className={styles.root}>
      <Spinner size='lg' />
    </div>
  )
}
