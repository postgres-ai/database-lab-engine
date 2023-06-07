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
  if (instance.sshServerUrl) {
    // Parse the URL to get the port
    const url = new URL(instance.url)
    const port = url.port || '2345'
    return `ssh -NTML ${port}:localhost:${port} ${instance.sshServerUrl} -i ~/.ssh/id_rsa`
  } else {
    return null
  }
}
