/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { observer } from 'mobx-react-lite'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { Link } from '@postgres.ai/shared/components/Link2'
import { ImportantText } from '@postgres.ai/shared/components/ImportantText'
import { Text } from '@postgres.ai/shared/components/Text'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'

type Props = {
  isOpen: boolean
  onClose: () => void
  cloneId: string
  clonePagePath?: string
}

export const DestroyCloneRestrictionModal = observer((props: Props) => {
  const { isOpen, onClose, cloneId, clonePagePath } = props

  return (
    <Modal title="Cannot delete clone" isOpen={isOpen} onClose={onClose}>
      <Text>
        Cannot delete clone <ImportantText>{cloneId}</ImportantText> because
        deletion protection is enabled. You can disable deletion protection on{' '}
        {clonePagePath ? (
          <Link to={clonePagePath}>the clone page</Link>
        ) : (
          'this page'
        )}
        .
      </Text>

      <SimpleModalControls
        items={[
          {
            text: 'Close',
            onClick: onClose,
          },
        ]}
      />
    </Modal>
  )
})
