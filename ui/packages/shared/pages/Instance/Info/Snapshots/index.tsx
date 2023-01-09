/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/styles'
import { observer } from 'mobx-react-lite'
import { formatDistanceToNowStrict } from 'date-fns'

import { Button } from '@postgres.ai/shared/components/Button2'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'

import { Section } from '../components/Section'
import { Property } from '../components/Property'

import { getEdgeSnapshots } from './utils'
import { Calendar } from './Calendar'

const useStyles = makeStyles(
  {
    button: {
      marginTop: '20px',
    },
  },
  { index: 1 },
)

export const Snapshots = observer(() => {
  const stores = useStores()

  const { snapshots } = stores.main

  const classes = useStyles()

  const { firstSnapshot, lastSnapshot } = getEdgeSnapshots(snapshots.data ?? [])

  return (
    <Section
      title="Snapshots"
      rightContent={
        !snapshots.data && !snapshots.error && <Spinner size="sm" />
      }
    >
      {snapshots.error && <ErrorStub {...snapshots.error} size="normal" />}

      {snapshots.data && !snapshots.error && (
        <>
          <Property name="Number of snapshots">
            {snapshots.data.length}
          </Property>

          <Property name="Oldest data state time">
            {lastSnapshot && (
              <>
                {lastSnapshot.dataStateAt} (
                {formatDistanceToNowStrict(lastSnapshot.dataStateAtDate, {
                  addSuffix: true,
                })}
                )
              </>
            )}
            {!lastSnapshot && '-'}
          </Property>

          <Property name="Newest data state time">
            {firstSnapshot && (
              <>
                {firstSnapshot.dataStateAt} (
                {formatDistanceToNowStrict(firstSnapshot.dataStateAtDate, {
                  addSuffix: true,
                })}
                )
              </>
            )}
            {!firstSnapshot && '-'}
          </Property>

          <Calendar
            snapshots={snapshots.data}
            onSelectDate={(date) => stores.snapshotsModal.openModal({ date })}
          />

          <Button
            className={classes.button}
            onClick={() => stores.snapshotsModal.openModal()}
          >
            Show all snapshots
          </Button>
        </>
      )}
    </Section>
  )
})
