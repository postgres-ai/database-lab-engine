/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/styles'
import { observer } from 'mobx-react-lite'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { ShieldIcon } from '@postgres.ai/shared/icons/Shield'
import { WarningIcon } from '@postgres.ai/shared/icons/Warning'

import { Section } from '../components/Section'
import { Property } from '../components/Property'
import { ValueStatus } from '../components/ValueStatus'

import { ConnectModal } from './ConnectModal'

const useStyles = makeStyles({
  connectButton: {
    marginTop: '10px',
  },
  url: {
    overflowWrap: 'break-word',
  },
  icon: {
    top: 0,
    left: 0,
    position: 'absolute',
    height: '100%',
    width: '100%',
  },
})

const checkIsSecureUrl = (urlStr: string) => {
  const url = new URL(urlStr)
  return url.protocol === 'https:'
}

export const Connection = observer(() => {
  const stores = useStores()
  const classes = useStyles()

  const { instance } = stores.main
  if (!instance?.url) return null

  const isSecureUrl = checkIsSecureUrl(instance.url) || instance.useTunnel

  return (
    <Section title="Connection">
      <Property name="URL">
        <span className={classes.url}>{instance.url}</span>
        <br />
        <ValueStatus
          type={isSecureUrl ? 'ok' : 'warning'}
          icon={
            isSecureUrl ? (
              <ShieldIcon className={classes.icon} />
            ) : (
              <WarningIcon className={classes.icon} />
            )
          }
        >
          The connection to Database Lab API is{' '}
          {isSecureUrl ? 'secure' : 'not secure'}
        </ValueStatus>
      </Property>

      <Property name="WS tunnels">
        {instance.useTunnel ? 'used' : 'not used'}
      </Property>

      <ConnectModal className={classes.connectButton} />
    </Section>
  )
})
