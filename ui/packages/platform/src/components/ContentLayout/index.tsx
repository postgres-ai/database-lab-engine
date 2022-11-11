/* eslint-disable react-hooks/rules-of-hooks */
/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { useRouteMatch } from 'react-router-dom'
import clsx from 'clsx'
import { observer } from 'mobx-react-lite'

import { ROUTES } from 'config/routes'
import settings from 'utils/settings'
import { bannersStore } from 'stores/banners'

import { DemoOrgNotice } from './DemoOrgNotice'
import { DeprecatedApiBanner } from './DeprecatedApiBanner'
import { Footer } from './Footer'

import styles from './styles.module.scss'

type Props = {
  children: React.ReactNode
}

export const ContentLayout = React.memo(observer((props: Props) => {
  const { children } = props

  const isOrgJoeInstance = Boolean(
    useRouteMatch(ROUTES.ORG.JOE_INSTANCES.JOE_INSTANCE.createPath()),
  )

  const isProjectJoeInstance = Boolean(
    useRouteMatch(ROUTES.ORG.PROJECT.JOE_INSTANCES.JOE_INSTANCE.createPath()),
  )

  const isDemoOrg = Boolean(useRouteMatch(`/${settings.demoOrgAlias}`))

  const isHiddenFooter = isOrgJoeInstance || isProjectJoeInstance

  return (
    <div className={styles.root}>
      {isDemoOrg && <DemoOrgNotice />}
      { bannersStore.isOpenDeprecatedApi && <DeprecatedApiBanner /> }

      <div className={styles.wrapper} id="content-container">
        <main
          className={clsx(styles.content, isHiddenFooter && styles.fullScreen)}
        >
          {children}
        </main>
        <Footer />
      </div>
    </div>
  )
}))
