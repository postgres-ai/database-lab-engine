/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */
import { GetSnapshots } from '@postgres.ai/shared/types/api/endpoints/getSnapshots'
import {
  SnapshotDto,
  formatSnapshotDto,
} from '@postgres.ai/shared/types/api/entities/snapshot'

import { request } from 'helpers/request'

export const getSnapshots: GetSnapshots = async (req) => {
  const params = new URLSearchParams()
  if (req.branchName) {
    params.append('branch', req.branchName)
  }
  if (req.dataset) {
    params.append('dataset', req.dataset)
  }
  const url = `/snapshots${params.toString() ? `?${params.toString()}` : ''}`;
  const response = await request(url);

  return {
    response: response.ok
      ? ((await response.json()) as SnapshotDto[]).map(formatSnapshotDto)
      : null,
    error: response.ok ? null : response,
  }
}
