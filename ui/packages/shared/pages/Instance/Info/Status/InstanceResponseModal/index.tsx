import { observer } from 'mobx-react-lite'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { FormattedText } from '@postgres.ai/shared/components/FormattedText'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'

import styles from './styles.module.scss'

type Props = {
  isOpen: boolean
  onClose: () => void
}

export const InstanceResponseModal = observer((props: Props) => {
  const stores = useStores()

  if (!stores.main.instance) return null

  return (
    <Modal
      isOpen={props.isOpen}
      onClose={props.onClose}
      title="Instance Full Response"
      size="md"
    >
      <FormattedText
        value={JSON.stringify(stores.main.instance.dto, null, 2)}
        className={styles.formattedText}
      />
    </Modal>
  )
})
