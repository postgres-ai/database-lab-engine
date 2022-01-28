/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export const getTags = ({
  pool,
  snapshotId,
}: {
  pool: string | null
  snapshotId: string | null
}) => {
  const tags = []

  if (pool) tags.push({ name: 'Disk', value: pool })
  if (snapshotId) tags.push({ name: 'Snapshot', value: snapshotId })

  return tags
}
