import cn from 'classnames'
import { Link } from 'react-router-dom'

import { ROUTES } from 'config/routes'

import styles from './styles.module.scss'
import { DLEEdition } from 'helpers/edition'
import { LogoIcon } from './icons'

type Props = {
  isCollapsed: boolean
}

export const Header = (props: Props) => {
  return (
    <header className={cn(styles.root, props.isCollapsed && styles.collapsed)}>
      <Link
        to={ROUTES.path}
        className={cn(styles.header, props.isCollapsed && styles.collapsed)}
      >
        <LogoIcon className={styles.logo} />

        {!props.isCollapsed && (
          <h1 className={styles.title}>
            Database Lab
            <br />
            <span className={styles.name}>{DLEEdition()}</span>
          </h1>
        )}
      </Link>
    </header>
  )
}
