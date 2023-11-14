/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { RouteComponentProps } from 'react-router'

import DbLabInstanceInstallForm from 'components/DbLabInstanceInstallForm/DbLabInstanceInstallForm'

import { useInstanceFormStyles } from 'components/DbLabInstanceForm/DbLabInstanceFormWrapper'

export interface DbLabInstanceFormProps {
  edit?: boolean
  orgId: number
  project: string | undefined
  history: RouteComponentProps['history']
  orgPermissions: {
    dblabInstanceCreate?: boolean
  }
}

export const DbLabInstanceFormInstallWrapper = (
  props: DbLabInstanceFormProps,
) => {

  const classes = useInstanceFormStyles()

  return <DbLabInstanceInstallForm {...props} classes={classes} />
}
