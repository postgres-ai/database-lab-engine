/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import ConsoleColors from '../styles/ConsoleColors';

export default {
  ce: {
    id: 'ce',
    name: 'CE',
    title: 'Postgres.ai Community Edition',
    color: ConsoleColors.secondary2.main,
    limits: {
      maxDblabInstances: 1,
      maxJoeInstances: 1,
      daysJoeHistory: 14,
      emailDomainRestricted: true
    }
  },
  ee_gold_monthly: {
    color: ConsoleColors.pgaiOrange
  }
};
