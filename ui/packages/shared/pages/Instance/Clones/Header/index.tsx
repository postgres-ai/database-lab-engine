/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { formatBytesIEC } from '@postgres.ai/shared/utils/units'

import { Item } from './Item'

import styles from './styles.module.scss'

type Props = {
  expectedCloningTimeS: number
  logicalSize: number | null
  clonesCount: number
  clonesCountLastMonth?: number
}

export const Header = (props: Props) => {
  const {
    expectedCloningTimeS,
    logicalSize,
    clonesCount,
    clonesCountLastMonth,
  } = props

  return (
    <div className={styles.root}>
      <Item value={expectedCloningTimeS ? `${expectedCloningTimeS} s` : '-'}>
        average
        <br />
        cloning time
      </Item>

      <Item
        value={
          logicalSize ? formatBytesIEC(logicalSize, { precision: 2 }) : '-'
        }
      >
        logical
        <br />
        data size
      </Item>

      <Item value={clonesCount}>
        clones
        <br />
        now
      </Item>

      <Item
        value={
          logicalSize
            ? formatBytesIEC(logicalSize * clonesCount, { precision: 2 })
            : '-'
        }
      >
        total
        <br />
        size of clones
      </Item>

      {clonesCountLastMonth && (
        <Item value={clonesCountLastMonth}>
          clones
          <br />
          in last month
        </Item>
      )}
    </div>
  )
}
