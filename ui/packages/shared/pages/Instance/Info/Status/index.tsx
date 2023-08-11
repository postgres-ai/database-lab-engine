/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import { observer } from 'mobx-react-lite'

import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'
import { Status as StatusIndicator } from '@postgres.ai/shared/components/Status'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { formatDateStd } from '@postgres.ai/shared/utils/date'
import { Button } from '@postgres.ai/shared/components/Button2'
import { linksConfig } from '@postgres.ai/shared/config/links'

import { Section } from '../components/Section'
import { Property } from '../components/Property'

import { InstanceResponseModal } from './InstanceResponseModal'

import { getType, getText } from './utils'

import styles from './styles.module.scss'

export const Status = observer(() => {
  const [isOpenInstanceResponseModal, setIsOpenInstanceResponseModal] =
    useState(false)
  const stores = useStores()

  const { instance } = stores.main
  if (!instance || !instance.state) return null

  const { code, message } = instance.state.status
  const { version, startedAt } = instance.state.engine

  const isStatusOk = code === 'OK'

  return (
    <Section title="Status">
      <Property name="Status">
        <StatusIndicator type={getType(code)}>
          {getText(code)}
          {!isStatusOk && (
            <>
              <br />
              {message}
            </>
          )}
        </StatusIndicator>
      </Property>
      {startedAt && (
        <Property name="Started">
          {formatDateStd(startedAt, { withDistance: true })}
        </Property>
      )}
      {instance.createdAt && (
        <Property name="Registered">
          {formatDateStd(instance.createdAt, { withDistance: true })}
        </Property>
      )}
      {version && <Property name="Version">{version}</Property>}

      {!isStatusOk && (
        <div className={styles.controls}>
          <Button
            onClick={() => setIsOpenInstanceResponseModal(true)}
            className={styles.button}
          >
            Show full response
          </Button>
          <GatewayLink className={styles.button} href={linksConfig.support}>
            <Button theme="accent">Ask support</Button>
          </GatewayLink>
        </div>
      )}

      <InstanceResponseModal
        isOpen={isOpenInstanceResponseModal}
        onClose={() => setIsOpenInstanceResponseModal(false)}
      />
    </Section>
  )
})
