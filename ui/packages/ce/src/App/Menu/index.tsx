import { useState } from 'react'
import cn from 'classnames'
import { observer } from 'mobx-react-lite'

import { linksConfig } from '@postgres.ai/shared/config/links'

import { Header } from './Header'
import { Button } from './components/Button'
import githubIconUrl from './icons/github.svg'
import docsIconUrl from './icons/docs.svg'
import discussionIconUrl from './icons/discussion.svg'
import arrowLeftIconUrl from './icons/arrow-left.svg'
import arrowRightIconUrl from './icons/arrow-right.svg'

import styles from './styles.module.scss'

const LAPTOP_WIDTH_PX = 1024

export const Menu = observer(() => {
  const [isCollapsed, setIsCollapsed] = useState(
    () => window.innerWidth < LAPTOP_WIDTH_PX,
  )

  return (
    <div className={cn(styles.root, isCollapsed && styles.collapsed)}>
      <div className={styles.content}>
        <Header isCollapsed={isCollapsed} />
      </div>
      <footer className={styles.footer}>
        <Button
          type="gateway-link"
          href={linksConfig.github}
          icon={<img src={githubIconUrl} alt="GitHub" />}
          isCollapsed={isCollapsed}
        >
          Star us on GitHub
        </Button>

        <Button
          type="gateway-link"
          href={linksConfig.docs}
          icon={<img src={docsIconUrl} alt="Documentation" />}
          isCollapsed={isCollapsed}
        >
          Documentation
        </Button>

        <Button
          type="gateway-link"
          href={linksConfig.support}
          className={styles.supportBtn}
          icon={<img src={discussionIconUrl} alt="Discussion" />}
          isCollapsed={isCollapsed}
        >
          Ask support
        </Button>

        <Button
          className={styles.collapseBtn}
          onClick={() => setIsCollapsed(!isCollapsed)}
          isCollapsed={isCollapsed}
          icon={
            <img
              src={isCollapsed ? arrowRightIconUrl : arrowLeftIconUrl}
              alt={isCollapsed ? 'Arrow right' : 'Arrow left'}
            />
          }
        >
          Collapse
        </Button>
      </footer>
    </div>
  )
})
