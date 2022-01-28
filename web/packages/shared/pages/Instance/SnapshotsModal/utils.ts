/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { formatUTC } from '@postgres.ai/shared/utils/date'

export const getTags = ({ date, pool }: { date: Date | null, pool: string | null }) => {
  const tags = []

  if (date) tags.push({ name: 'Date', value: `${formatUTC(date, 'yyyy-MM-dd')} UTC` })
  if (pool) tags.push({ name: 'Disk', value: pool })

  return tags
}
