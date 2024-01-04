/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useHistory } from 'react-router'
import { observer } from 'mobx-react-lite'
import { makeStyles } from '@material-ui/core'

import { useStores, useHost } from '@postgres.ai/shared/pages/Instance/context'
import { SnapshotsTable } from '@postgres.ai/shared/pages/Instance/Snapshots/components/SnapshotsTable'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { isSameDayUTC } from '@postgres.ai/shared/utils/date'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { Button } from '@postgres.ai/shared/components/Button2'
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
    spinner: {
      position: 'absolute',
      right: '50%',
      transform: 'translate(-50%, -50%)',
    },
  },
  { index: 1 },
)

export const Snapshots = observer(() => {
  const host = useHost()
  const stores = useStores()
  const classes = useStyles()
  const history = useHistory()

  const { snapshots, instance } = stores.main

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
  const goToSnapshotAddPage = () => history.push(host.routes.createSnapshot())

  if (!instance && !snapshots.isLoading) return <></>

  if (snapshots?.error)
    return (
      <ErrorStub
        title={snapshots?.error?.title}
        message={snapshots?.error?.message}
      />
    )

  return (
    <div className={classes.marginTop}>
      {snapshots.isLoading ? (
        <Spinner size="lg" className={classes.spinner} />
      ) : (
        <>
          <SectionTitle
            level={2}
            tag="h2"
            text={`Snapshots (${filteredSnapshots?.length || 0})`}
            rightContent={
              <>
                <Button
                  theme="primary"
                  onClick={goToSnapshotAddPage}
                  isDisabled={!hasClones}
                >
                  Create snapshot
                </Button>

                {!hasClones && (
                  <Tooltip content="No clones">
                    <div>
                      <InfoIcon className={classes.infoIcon} />
                    </div>
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
        </>
      )}
    </div>
  )
})
