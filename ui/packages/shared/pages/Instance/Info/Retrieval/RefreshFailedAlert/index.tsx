import { observer } from 'mobx-react-lite'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { WarningIcon } from '@postgres.ai/shared/icons/Warning'
import { formatDateStd } from '@postgres.ai/shared/utils/date'

import { Property } from '../../components/Property'

import styles from './styles.module.scss'

export const RefreshFailedAlert = observer(() => {
  const stores = useStores()

  const refreshFailed =
    stores.main.instance?.state.retrieving?.alerts?.refreshFailed
  if (!refreshFailed) return null

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <h6 className={styles.title}>{refreshFailed.message}</h6>
        <WarningIcon className={styles.icon} />
      </div>
      <Property classes={{ name: styles.propertyName }} name="Last seen">
        {formatDateStd(refreshFailed.lastSeen, { withDistance: true })}
      </Property>
      <Property classes={{ name: styles.propertyName }} name="Count">
        {refreshFailed.count}
      </Property>
    </div>
  )
})
