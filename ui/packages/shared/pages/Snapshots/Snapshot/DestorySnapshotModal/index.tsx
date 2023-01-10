/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useEffect, useState } from 'react'
import { makeStyles } from '@material-ui/core'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { ImportantText } from '@postgres.ai/shared/components/ImportantText'
import { Text } from '@postgres.ai/shared/components/Text'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'

type Props = {
  snapshotId: string
  isOpen: boolean
  onClose: () => void
  onDestroySnapshot: () => void
  destroySnapshotError: { title?: string; message: string } | null
}

const useStyles = makeStyles(
  {
    errorMessage: {
      color: 'red',
      marginTop: '10px',
    },
  },
  { index: 1 },
)

export const DestroySnapshotModal = ({
  snapshotId,
  isOpen,
  onClose,
  onDestroySnapshot,
  destroySnapshotError,
}: Props) => {
  const classes = useStyles()
  const [deleteError, setDeleteError] = useState(destroySnapshotError?.message)

  const handleClickDestroy = () => {
    onDestroySnapshot()
  }

  const handleClose = () => {
    setDeleteError('')
    onClose()
  }

  useEffect(() => {
    setDeleteError(destroySnapshotError?.message)
  }, [destroySnapshotError])

  return (
    <Modal title={'Confirmation'} onClose={handleClose} isOpen={isOpen}>
      <Text>
        Are you sure you want to destroy snapshot{' '}
        <ImportantText>{snapshotId}</ImportantText>?
      </Text>
      {deleteError && <p className={classes.errorMessage}>{deleteError}</p>}
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
