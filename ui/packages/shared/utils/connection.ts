/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Instance } from '@postgres.ai/shared/types/api/entities/instance'
import { Clone } from '@postgres.ai/shared/types/api/entities/clone'

const isEmptyDbData = (clone: Clone) => {
  const { host, port, username } = clone.db

  // 'host', 'port' or 'username' can be empty string.
  return !host || !port || !username
}

export const getSshPortForwardingCommand = (
  instance: Instance,
  clone: Clone,
) => {
  if (!instance.sshServerUrl) return null
  if (isEmptyDbData(clone)) return null

  return `ssh -NTML ${clone.db.port}:localhost:${clone.db.port} ${instance.sshServerUrl} -i ~/.ssh/id_rsa`
}

export const getPsqlConnectionStr = (clone: Clone) => {
  if (isEmptyDbData(clone)) return null

  return `"host=${clone.db.host} port=${clone.db.port} user=${clone.db.username} dbname=DBNAME"`
}

export const getJdbcConnectionStr = (clone: Clone) => {
  if (isEmptyDbData(clone)) return null

  return `jdbc:postgresql://${clone.db.host}:${clone.db.port}/DBNAME?user=${clone.db.username}&password=DBPASSWORD`
}
