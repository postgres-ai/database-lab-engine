/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/core'
import clsx from 'clsx'

import {
  CloneDto,
  formatCloneDto,
} from '@postgres.ai/shared/types/api/entities/clone'
import { Status as StatusBase } from '@postgres.ai/shared/components/Status'
import { FormattedText } from '@postgres.ai/shared/components/FormattedText'
import {
  getCloneStatusType,
  getCloneStatusText,
} from '@postgres.ai/shared/utils/clone'

type Props = {
  rawClone: CloneDto
  className?: string
}

const useStyles = makeStyles(
  {
    root: {
      marginTop: '2px',
    },
    status: {
      fontWeight: 500,
    },
    message: {
      margin: '4px 0 0 0',
    },
    errorMessage: {
      marginTop: '8px',
    },
  },
  { index: 1 },
)

export const Status = React.memo((props: Props) => {
  const { rawClone, className } = props

  const classes = useStyles()

  const clone = formatCloneDto(rawClone)

  const { code, message } = clone.status

  const statusType = getCloneStatusType(code)
  const statusText = getCloneStatusText(code)

  const isError = statusType === 'error'

  return (
    <div className={clsx(classes.root, className)}>
      <StatusBase type={statusType} className={classes.status}>
        {statusText}
      </StatusBase>

      {!isError && <p className={classes.message}>{message}</p>}

      {isError && (
        <FormattedText value={message} className={classes.errorMessage} />
      )}
    </div>
  )
})
