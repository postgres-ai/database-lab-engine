import { useState } from 'react'
import cn from 'classnames'
import { observer } from 'mobx-react-lite'

import { linksConfig } from '@postgres.ai/shared/config/links'
import { Button } from '@postgres.ai/shared/components/MenuButton'
import { ROUTES } from 'config/routes'

import { SignOutModal } from './SignOutModal'
import { Header } from './Header'
import {
  ArrowRight,
  ArrowLeft,
  Discussion,
  Docs,
  ExitIcon,
  Github,
} from './icons'

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
            icon={<Github />}
            isCollapsed={isCollapsed}
          >
            Star us on GitHub
          </Button>

          <Button
            type="gateway-link"
            href={linksConfig.docs}
            icon={<Docs />}
            isCollapsed={isCollapsed}
          >
            Documentation
          </Button>

          <Button
            type="gateway-link"
            href={linksConfig.support}
            className={styles.supportBtn}
            icon={<Discussion />}
            isCollapsed={isCollapsed}
          >
            Ask support
          </Button>
          {isValidToken && (
            <Button
              type="button"
              onClick={() => setIsOpen(true)}
              icon={<ExitIcon />}
              isCollapsed={isCollapsed}
            >
              Sign out
            </Button>
          )}
          <Button
            className={styles.collapseBtn}
            onClick={handleCollapse}
            isCollapsed={isCollapsed}
            icon={isCollapsed ? <ArrowRight /> : <ArrowLeft />}
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
