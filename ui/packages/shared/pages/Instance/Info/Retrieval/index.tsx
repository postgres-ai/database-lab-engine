import { observer } from 'mobx-react-lite'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { Status } from '@postgres.ai/shared/components/Status'
import { capitalize } from '@postgres.ai/shared/utils/strings'
import { formatDateStd } from '@postgres.ai/shared/utils/date'

import { Section } from '../components/Section'
import { Property } from '../components/Property'

import { RefreshFailedAlert } from './RefreshFailedAlert'

import { getTypeByStatus } from './utils'

export const Retrieval = observer(() => {
  const stores = useStores()

  const { instance } = stores.main
  if (!instance) return null

  const { retrieving } = instance.state
  if (!retrieving) return null

  return (
    <Section title="Retrieval">
      <Property name="Status">
        <Status type={getTypeByStatus(retrieving.status)}>
          {capitalize(retrieving.status)}
        </Status>
      </Property>
      <Property name="Mode">{retrieving.mode}</Property>
      <Property name="Last refresh">
        {retrieving.lastRefresh
          ? formatDateStd(retrieving.lastRefresh, { withDistance: true })
          : '-'}
      </Property>
      <Property name="Next refresh">
        {retrieving.nextRefresh
          ? formatDateStd(retrieving.nextRefresh, { withDistance: true })
          : 'Not scheduled'}
      </Property>
      <RefreshFailedAlert />
    </Section>
  )
})
