/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { ImportantText } from '@postgres.ai/shared/components/ImportantText'
import { Text } from '@postgres.ai/shared/components/Text'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'

type Props = {
  cloneId: string
  isOpen: boolean
  onClose: () => void
  onDestroyClone: () => void
}

export const DestroyCloneModal = (props: Props) => {
  const { cloneId, isOpen, onClose, onDestroyClone } = props

  const handleClickDestroy = () => {
    onDestroyClone()
    onClose()
  }

  return (
    <Modal
      title={`Destroy clone ${cloneId}`}
      onClose={onClose}
      isOpen={isOpen}
    >
      <Text>
        Are you sure you want to destroy clone{' '}
        <ImportantText>{cloneId}</ImportantText>?
      </Text>
      
      <SimpleModalControls
        items={[
          {
            text: 'Cancel',
            onClick: onClose,
          },
          {
            text: 'Destroy clone',
            variant: 'primary',
            onClick: handleClickDestroy,
          }
        ]}
      />
    </Modal>
  )
}
