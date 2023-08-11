/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import { observer } from 'mobx-react-lite'

import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { TableBodyCellMenu } from '@postgres.ai/shared/components/Table'

import { DestroyCloneModal } from '@postgres.ai/shared/components/DestroyCloneModal'
import { DestroyCloneRestrictionModal } from '@postgres.ai/shared/components/DestroyCloneRestrictionModal'
import { ResetCloneModal } from '@postgres.ai/shared/components/ResetCloneModal'

type Props = {
  clone: Clone
  onConnect: (cloneId: string) => void
  clonePagePath: string
}

export const MenuCell = observer((props: Props) => {
  const { clone, onConnect, clonePagePath } = props

  const stores = useStores()

  const [
    isOpenCloneDestroyRestrictionModal,
    setIsOpenCloneDestroyRestrictionModal,
  ] = useState(false)

  const [isOpenCloneDestroyModal, setIsOpenCloneDestroyModal] = useState(false)

  const [isOpenResetCloneModal, setIsOpenResetCloneModal] = useState(false)

  const handleClickDestroyClone = () => {
    if (clone.protected) return setIsOpenCloneDestroyRestrictionModal(true)
    setIsOpenCloneDestroyModal(true)
  }

  const handleClickResetClone = async () => {
    setIsOpenResetCloneModal(true)
  }

  const isLoading = stores.main.unstableClones.has(clone.id)

  if (!stores.main.instance) return null

  const hasSnapshots = Boolean(stores.main.snapshots.data?.length)

  return (
    <TableBodyCellMenu
      isLoading={isLoading}
      isDisabled={stores.main.isDisabledInstance}
      actions={[
        {
          name: 'Connect',
          onClick: () => onConnect(clone.id),
        },
        {
          name: 'Destroy clone',
          onClick: handleClickDestroyClone,
        },
        {
          name: 'Reset clone',
          onClick: handleClickResetClone,
          isDisabled: !hasSnapshots,
        },
      ]}
    >
      <DestroyCloneRestrictionModal
        cloneId={clone.id}
        isOpen={isOpenCloneDestroyRestrictionModal}
        onClose={() => setIsOpenCloneDestroyRestrictionModal(false)}
        clonePagePath={clonePagePath}
      />
      <DestroyCloneModal
        cloneId={clone.id}
        isOpen={isOpenCloneDestroyModal}
        onClose={() => setIsOpenCloneDestroyModal(false)}
        onDestroyClone={() => stores.main.destroyClone(clone.id)}
      />
      <ResetCloneModal
        clone={clone}
        snapshots={stores.main.snapshots.data}
        isOpen={isOpenResetCloneModal}
        onClose={() => setIsOpenResetCloneModal(false)}
        onResetClone={(snapshotId) =>
          stores.main.resetClone(clone.id, snapshotId)
        }
        version={stores.main.instance.state?.engine.version}
      />
    </TableBodyCellMenu>
  )
})
