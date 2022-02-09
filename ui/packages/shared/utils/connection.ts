/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

 import { Instance } from '@postgres.ai/shared/types/api/entities/instance'
 import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
 
 export const getSshPortForwardingCommand = (
   instance: Instance,
   clone: Clone,
 ) => {
   if (!instance.sshServerUrl) return null
   return `ssh -NTML ${clone.db.port}:localhost:${clone.db.port} ${instance.sshServerUrl} -i ~/.ssh/id_rsa`
 }
 
 export const getPsqlConnectionStr = (clone: Clone) =>
   `"host=${clone.db.host} port=${clone.db.port} user=${clone.db.username} dbname=DBNAME"`
 
 export const getJdbcConnectionStr = (clone: Clone) =>
   `jdbc:postgresql://${clone.db.host}:${clone.db.port}/DBNAME?user=${clone.db.username}&password=DBPASSWORD`
 