/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { colors } from '@postgres.ai/shared/styles/colors'

export const plans = {
  ce: {
    id: 'ce',
    name: 'CE',
    title: 'Postgres.ai Community Edition',
    color: colors.secondary2.main,
    limits: {
      maxDblabInstances: 1,
      maxJoeInstances: 1,
      daysJoeHistory: 14,
      emailDomainRestricted: true,
    },
  },
  ee_gold_monthly: {
    color: colors.pgaiOrange,
  },
}
