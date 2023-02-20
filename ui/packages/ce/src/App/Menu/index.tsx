import { useState } from 'react'
import cn from 'classnames'
import { observer } from 'mobx-react-lite'

import { linksConfig } from '@postgres.ai/shared/config/links'
import { Button } from '@postgres.ai/shared/components/MenuButton'
import { ROUTES } from 'config/routes'

import { SignOutModal } from './SignOutModal'
import { Header } from './Header'
import githubIconUrl from './icons/github.svg'
import docsIconUrl from './icons/docs.svg'
import exitIcon from './icons/exit-icon.svg'
import discussionIconUrl from './icons/discussion.svg'
import arrowLeftIconUrl from './icons/arrow-left.svg'
import arrowRightIconUrl from './icons/arrow-right.svg'

import styles from './styles.module.scss'

const LAPTOP_WIDTH_PX = 1024
const SIDEBAR_COLLAPSED_PARAM = 'sidebarMenuCollapsed'

export const Menu = observer(
  ({ isValidToken }: { isValidToken: boolean | undefined }) => {
    const [isOpen, setIsOpen] = useState(false)
    const [isCollapsed, setIsCollapsed] = useState(
      () =>
        window.innerWidth < LAPTOP_WIDTH_PX ||
        localStorage.getItem(SIDEBAR_COLLAPSED_PARAM) === '1',
    )

    const handleCollapse = () => {
      setIsCollapsed(!isCollapsed)
      localStorage.setItem(SIDEBAR_COLLAPSED_PARAM, isCollapsed ? '0' : '1')
    }

    const handleSignOut = () => {
      localStorage.removeItem('token')
      window.location.href = ROUTES.AUTH.path
    }

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
          {isValidToken && (
            <Button
              type="button"
              onClick={() => setIsOpen(true)}
              icon={<img src={exitIcon} alt="Profile" />}
              isCollapsed={isCollapsed}
            >
              Sign out
            </Button>
          )}
          <Button
            className={styles.collapseBtn}
            onClick={handleCollapse}
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
        {isOpen && (
          <SignOutModal
            handleSignOut={handleSignOut}
            onClose={() => setIsOpen(false)}
            isOpen={isOpen}
          />
        )}
      </div>
    )
  },
)
