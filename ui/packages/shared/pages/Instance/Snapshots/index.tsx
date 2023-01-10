/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import { observer } from 'mobx-react-lite'
import { makeStyles } from '@material-ui/core'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { SnapshotsTable } from '@postgres.ai/shared/pages/Instance/Snapshots/components/SnapshotsTable'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { isSameDayUTC } from '@postgres.ai/shared/utils/date'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinner'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { Button } from '@postgres.ai/shared/components/Button2'
import { CreateSnapshotModal } from '@postgres.ai/shared/pages/Instance/Snapshots/components/CreateSnapshotModal'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { InfoIcon } from '@postgres.ai/shared/icons/Info'

const useStyles = makeStyles(
  {
    marginTop: {
      marginTop: '16px',
    },
    infoIcon: {
      height: '12px',
      width: '12px',
      marginLeft: '8px',
      color: '#808080',
    },
  },
  { index: 1 },
)

export const Snapshots = observer(() => {
  const stores = useStores()
  const classes = useStyles()

  const [isCreateSnapshotOpen, setIsCreateSnapshotOpen] = useState(false)

  const { snapshots, instance, createSnapshot, createSnapshotError } =
    stores.main

  const filteredSnapshots =
    snapshots?.data &&
    snapshots.data.filter((snapshot) => {
      const isMatchedByDate =
        !stores.snapshotsModal.date ||
        isSameDayUTC(snapshot.dataStateAtDate, stores.snapshotsModal.date)

      const isMatchedByPool =
        !stores.snapshotsModal.pool ||
        snapshot.pool === stores.snapshotsModal.pool

      return isMatchedByDate && isMatchedByPool
    })

  const clonesList = instance?.state?.cloning.clones || []
  const isEmpty = !filteredSnapshots?.length
  const hasClones = Boolean(clonesList?.length)

  if (!instance && !snapshots.isLoading) return <></>

  if (snapshots?.error)
    return (
      <ErrorStub
        title={snapshots?.error?.title}
        message={snapshots?.error?.message}
      />
    )

  if (snapshots.isLoading) return <StubSpinner />

  return (
    <div className={classes.marginTop}>
      <SectionTitle
        level={2}
        tag="h2"
        text={`Snapshots (${filteredSnapshots?.length || 0})`}
        rightContent={
          <>
            <Button
              theme="primary"
              onClick={() => setIsCreateSnapshotOpen(true)}
              isDisabled={!hasClones}
            >
              Create snapshot
            </Button>

            {!hasClones && (
              <Tooltip content="No clones">
                <InfoIcon className={classes.infoIcon} />
              </Tooltip>
            )}
          </>
        }
      />
      {!isEmpty ? (
        <SnapshotsTable />
      ) : (
        <p className={classes.marginTop}>
          This instance has no active snapshots
        </p>
      )}
      <CreateSnapshotModal
        isOpen={isCreateSnapshotOpen}
        onClose={() => setIsCreateSnapshotOpen(false)}
        createSnapshot={createSnapshot}
        createSnapshotError={createSnapshotError}
        clones={clonesList}
      />
    </div>
  )
})
