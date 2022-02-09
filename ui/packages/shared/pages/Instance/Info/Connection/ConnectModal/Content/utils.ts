/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */


import { Instance } from '@postgres.ai/shared/types/api/entities/instance'

export const getCliInitCommand = (instance: Instance) =>
  `dblab init --url ${instance.url} --token TOKEN  --environment-id ${instance.projectName}`

export const getSshPortForwardingCommand = (instance: Instance) => {
  if (!instance.sshServerUrl) return null
  return `ssh -NTML 2348:localhost:2345 ${instance.sshServerUrl} -i ~/.ssh/id_rsa`
}
