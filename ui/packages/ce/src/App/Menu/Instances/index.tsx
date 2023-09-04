import { observer } from 'mobx-react-lite'

import { ROUTES } from 'config/routes'

import { ReactComponent as PlusIcon } from './icons/plus.svg'
import { Button } from '@postgres.ai/shared/components/MenuButton'

import styles from './styles.module.scss'

type Props = {
  isCollapsed: boolean
}

export const Instances = observer((props: Props) => {
  return (
    <div className={styles.root}>
      <nav className={styles.links}>
        <Button
          type="link"
          to={ROUTES.INSTANCE.path}
          activeClassName={styles.selected}
          className={styles.link}
        >
          DBLab #1
        </Button>
      </nav>
      <Button
        activeClassName={styles.selected}
        // to={ROUTES.INSTANCES.ADD.path}
        // type="link"
        icon={<PlusIcon />}
        isCollapsed={props.isCollapsed}
        className={styles.addInstanceBtn}
      >
        Add instance
      </Button>
    </div>
  )
})
