/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { RouteComponentProps } from 'react-router'

import PostgresClusterInstallForm from './PostgresClusterInstallForm'
import { useInstanceFormStyles } from 'components/DbLabInstanceForm/DbLabInstanceFormWrapper'

export interface PostgresClusterInstallFormWrapperProps {
  edit?: boolean
  orgId: number
  project: string | undefined
  history: RouteComponentProps['history']
  orgPermissions: {
    dblabInstanceCreate?: boolean
  }
}

export const PostgresClusterInstallWrapper = (
  props: PostgresClusterInstallFormWrapperProps,
) => {
  const classes = useInstanceFormStyles()

  return <PostgresClusterInstallForm {...props} classes={classes} />
}
