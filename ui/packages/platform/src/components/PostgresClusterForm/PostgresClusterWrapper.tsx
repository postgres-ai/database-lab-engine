/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

 import { RouteComponentProps } from 'react-router'
 
 
import { useInstanceFormStyles } from 'components/DbLabInstanceForm/DbLabInstanceFormWrapper'
import PostgresCluster from './PostgresCluster'
 
 export interface PostgresClusterWrapperProps {
   edit?: boolean
   orgId: number
   project: string | undefined
   history: RouteComponentProps['history']
   orgPermissions: {
     dblabInstanceCreate?: boolean
   }
 }
 
 export const PostgresClusterWrapper = (props: PostgresClusterWrapperProps) => {
 
   const classes = useInstanceFormStyles()
 
   return <PostgresCluster {...props} classes={classes} />
 }
 