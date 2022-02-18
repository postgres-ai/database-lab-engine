import { observer } from 'mobx-react-lite'

import { icons } from '@postgres.ai/shared/styles/icons'
import { Button } from '@postgres.ai/shared/components/Button'

import { appStore } from 'stores/app'

import styles from './styles.module.scss'

export const AppUpdateBanner = observer(() => {
  if (!appStore.isOutdatedVersion) return null

  return (
    <div className={styles.root}>
      <div className={styles.text}>
        {icons.updateIcon}&nbsp;UI update is available
      </div>
      <Button
        variant="primary"
        onClick={() => window.location.reload()}
      >
        Apply update
      </Button>
    </div>
  )
})
