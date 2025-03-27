/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import {
  Checkbox,
  FormControlLabel,
  Typography,
  makeStyles,
} from '@material-ui/core'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { ImportantText } from '@postgres.ai/shared/components/ImportantText'
import { Text } from '@postgres.ai/shared/components/Text'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'
import { DestroySnapshot } from '@postgres.ai/shared/types/api/endpoints/destroySnapshot'

type Props = {
  snapshotId: string
  instanceId: string
  isOpen: boolean
  onClose: () => void
  afterSubmitClick: () => void
  destroySnapshot: DestroySnapshot
}

interface ErrorResponse {
  error?: {
    message?: string
    details?: string
  }
}

const useStyles = makeStyles(
  {
    errorMessage: {
      color: 'red',
      marginTop: '10px',
      wordBreak: 'break-all',
    },
    checkboxRoot: {
      padding: '9px 10px',
    },
    grayText: {
      color: '#8a8a8a',
      fontSize: '12px',
      wordBreak: 'break-word',
    },
    marginTop: {
      marginTop: '6px',
    },
  },
  { index: 1 },
)

export const DestroySnapshotModal = ({
  snapshotId,
  instanceId,
  isOpen,
  onClose,
  afterSubmitClick,
  destroySnapshot,
}: Props) => {
  const classes = useStyles()
  const [forceDelete, setForceDelete] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [isForceDeleteOptionVisible, setForceDeleteOptionVisible] =
    useState(false)

  const handleClose = () => {
    setDeleteError(null)
    onClose()
  }

  const handleClickDestroy = () => {
    destroySnapshot(snapshotId, forceDelete, instanceId).then((res) => {
      if (res?.error) {
        const errorMessage =
          (res as ErrorResponse)?.error?.message ||
          (res as ErrorResponse)?.error?.details
        setDeleteError(errorMessage || null)
        setForceDeleteOptionVisible(true)
      } else {
        afterSubmitClick()
        handleClose()
      }
    })
  }

  return (
    <Modal
      title={'Confirmation'}
      onClose={handleClose}
      isOpen={isOpen}
      size="sm"
    >
      <Text>
        Are you sure you want to destroy snapshot{' '}
        <ImportantText>{snapshotId}</ImportantText>? This action cannot be
        undone.
      </Text>
      {deleteError && <p className={classes.errorMessage}>{deleteError}</p>}
      {isForceDeleteOptionVisible && (
        <div className={classes.marginTop}>
          <FormControlLabel
            control={
              <Checkbox
                name="debug"
                checked={forceDelete}
                onChange={(e) => setForceDelete(e.target.checked)}
                classes={{
                  root: classes.checkboxRoot,
                }}
              />
            }
            label={'Force delete'}
          />
          <Typography className={classes.grayText}>
            If the snapshot cannot be deleted due to dependencies, enabling
            “Force delete” will remove it along with all dependent snapshots and
            clones.
          </Typography>
        </div>
      )}
      <SimpleModalControls
        items={[
          {
            text: 'Cancel',
            onClick: handleClose,
          },
          {
            text: 'Destroy snapshot',
            variant: 'primary',
            onClick: handleClickDestroy,
          },
        ]}
      />
    </Modal>
  )
}
