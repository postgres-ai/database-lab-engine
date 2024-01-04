/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useEffect, useState } from 'react'
import { makeStyles } from '@material-ui/core'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { ModalProps } from '@postgres.ai/shared/pages/Branches/components/Modals/types'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'
import { ImportantText } from '@postgres.ai/shared/components/ImportantText'
import { Text } from '@postgres.ai/shared/components/Text'
interface DeleteBranchModalProps extends ModalProps {
  deleteBranchError: { title?: string; message?: string } | null
  deleteBranch: (branchName: string) => void
  branchName: string
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

export const DeleteBranchModal = ({
  isOpen,
  onClose,
  deleteBranchError,
  deleteBranch,
  branchName,
}: DeleteBranchModalProps) => {
  const classes = useStyles()
  const [deleteError, setDeleteError] = useState(deleteBranchError?.message)

  const handleSubmit = () => {
    deleteBranch(branchName)
  }

  const handleClose = () => {
    setDeleteError('')
    onClose()
  }

  useEffect(() => {
    setDeleteError(deleteBranchError?.message)
  }, [deleteBranchError])

  return (
    <Modal title="Confirmation" onClose={handleClose} isOpen={isOpen} size="xs">
      <Text>
        Are you sure you want to destroy branch{' '}
        <ImportantText>{branchName}</ImportantText>?
      </Text>
      {deleteError && <p className={classes.errorMessage}>{deleteError}</p>}
      <SimpleModalControls
        items={[
          {
            text: 'Cancel',
            onClick: handleClose,
          },
          {
            text: 'Destroy branch',
            variant: 'primary',
            onClick: handleSubmit,
            isDisabled: branchName === 'main',
          },
        ]}
      />
    </Modal>
  )
}
