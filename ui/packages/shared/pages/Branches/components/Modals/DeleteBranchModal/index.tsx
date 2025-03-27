/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import { makeStyles } from '@material-ui/core'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { ModalProps } from '@postgres.ai/shared/pages/Branches/components/Modals/types'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'
import { ImportantText } from '@postgres.ai/shared/components/ImportantText'
import { Text } from '@postgres.ai/shared/components/Text'
import { DeleteBranch } from '@postgres.ai/shared/types/api/endpoints/deleteBranch'
interface DeleteBranchModalProps extends ModalProps {
  deleteBranch: DeleteBranch
  branchName: string
  instanceId: string
  afterSubmitClick: () => void
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
  deleteBranch,
  branchName,
  instanceId,
  afterSubmitClick,
}: DeleteBranchModalProps) => {
  const classes = useStyles()
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const handleDelete = async () => {
    const deleteRes = await deleteBranch(branchName, instanceId)

    if (deleteRes?.error) {
      setDeleteError(deleteRes.error?.message)
    } else {
      afterSubmitClick()
    }
  }

  const handleClose = () => {
    setDeleteError(null)
    onClose()
  }

  return (
    <Modal title="Confirmation" onClose={handleClose} isOpen={isOpen} size="xs">
      <Text>
        Are you sure you want to destroy branch{' '}
        <ImportantText>{branchName}</ImportantText>? This action cannot be
        undone.
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
            onClick: handleDelete,
            isDisabled: branchName === 'main',
          },
        ]}
      />
    </Modal>
  )
}
