import { useState } from 'react'
import { observer } from 'mobx-react-lite'
import { makeStyles } from '@material-ui/core'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { Status } from '@postgres.ai/shared/components/Status'
import { capitalize } from '@postgres.ai/shared/utils/strings'
import { formatDateStd } from '@postgres.ai/shared/utils/date'
import { Button } from '@postgres.ai/shared/components/Button2'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { InfoIcon } from '@postgres.ai/shared/icons/Info'
import { isRetrievalUnknown } from '@postgres.ai/shared/pages/Configuration/utils'

import { Section } from '../components/Section'
import { Property } from '../components/Property'

import { RefreshFailedAlert } from './RefreshFailedAlert'

import { getTypeByStatus } from './utils'
import { RetrievalModal } from './RetrievalModal'

const useStyles = makeStyles(
  () => ({
    infoIcon: {
      height: '12px',
      width: '12px',
      marginLeft: '8px',
      color: '#808080',
    },
    detailsButton: {
      marginLeft: '8px',
    },
  }),
  { index: 1 },
)

export const Retrieval = observer(() => {
  const stores = useStores()
  const classes = useStyles()
  const [isModalOpen, setIsModalOpen] = useState<boolean>(false)

  const { instance, instanceRetrieval } = stores.main
  if (!instance) return null

  const { retrieving } = instance.state
  if (!retrieving) return null

  if (!instanceRetrieval) return null
  const { mode, status, activity } = instanceRetrieval
  const isVisible = mode !== 'physical' && !isRetrievalUnknown(mode)
  const isActive = mode === 'logical' && status === 'refreshing'

  return (
    <Section title="Retrieval">
      <Property name="Status">
        <Status type={getTypeByStatus(retrieving.status)}>
          {capitalize(retrieving.status)}
          {isVisible && (
            <>
              <Button
                theme="primary"
                onClick={() => setIsModalOpen(true)}
                isDisabled={!isActive}
                className={classes.detailsButton}
              >
                Show details
              </Button>
              {!isActive && (
                <Tooltip content="No retrieval activity details">
                  <InfoIcon className={classes.infoIcon} />
                </Tooltip>
              )}
            </>
          )}
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
      <RetrievalModal
        data={activity}
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </Section>
  )
})
