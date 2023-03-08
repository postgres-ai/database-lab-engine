import { useEffect } from 'react'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { ModalReloadButton } from '@postgres.ai/shared/pages/Instance/components/ModalReloadButton'
import { ActivityType } from '@postgres.ai/shared/types/api/entities/instanceRetrieval'
import { RetrievalTable } from '../RetrievalTable'

import styles from './styles.module.scss'

export const RetrievalModal = ({
  isOpen,
  onClose,
  data,
}: {
  isOpen: boolean
  onClose: () => void
  data: {
    source: ActivityType[]
    target: ActivityType[]
  }
}) => {
  const stores = useStores()
  const { isReloadingInstanceRetrieval, reloadInstanceRetrieval } = stores.main

  useEffect(() => {
    reloadInstanceRetrieval()
  },[])

  return (
    <Modal
      title="Retrieval activity details"
      isOpen={isOpen}
      onClose={onClose}
      size="md"
      titleRightContent={
        <ModalReloadButton
          isReloading={isReloadingInstanceRetrieval}
          onReload={reloadInstanceRetrieval}
        />
      }
    >
      <div className={styles.tableContainer}>
        <RetrievalTable data={data?.source} activity="source" />
        <RetrievalTable data={data?.target} activity="target" />
      </div>
    </Modal>
  )
}
