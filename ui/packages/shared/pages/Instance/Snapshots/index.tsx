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
import { SnapshotsList } from '@postgres.ai/shared/pages/Instance/Snapshots/components/SnapshotsList'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { Button } from '@postgres.ai/shared/components/Button2'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { InfoIcon } from '@postgres.ai/shared/icons/Info'
import { useEffect, useMemo, useState } from 'react'
import { Branch } from 'types/api/endpoints/getBranches'
import { SnapshotHeader } from './components/SnapshotHeader'

const useStyles = makeStyles(
  {
    sectionTitle: {
      borderBottom: 0,
    },
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
  const { getBranches, instance, snapshots } = stores.main
  const [messageFilter, setMessageFilter] = useState('')
  const [branches, setBranches] = useState<string[] | null>(null)
  const [selectedBranch, setSelectedBranch] = useState<string>()
  const [isLoadingBranches, setIsLoadingBranches] = useState(true)

  const filteredSnapshots = useMemo(() => {
    if (!snapshots.data) return []

    return snapshots.data.filter((snapshot) =>
      snapshot.message.toLowerCase().includes(messageFilter.toLowerCase()),
    )
  }, [snapshots.data, messageFilter])

  const clonesList = instance?.state?.cloning.clones || []
  const isEmpty = !filteredSnapshots?.length
  const hasClones = Boolean(clonesList?.length)
  const goToSnapshotAddPage = () => history.push(host.routes.createSnapshot())

  useEffect(() => {
    const fetchInitialData = async () => {
      try {
        setIsLoadingBranches(true)
        const branches = await getBranches()
        const branchNames = branches?.map(({ name }: Branch) => name) ?? []
        setBranches(branchNames)
      } catch (error) {
        console.error('Error fetching initial data:', error)
      } finally {
        setIsLoadingBranches(false)
        setSelectedBranch('main')
      }
    }

    fetchInitialData()
  }, [])

  useEffect(() => {
    if (selectedBranch) {
      stores.main.reloadSnapshots(selectedBranch)
    }
  }, [selectedBranch])

  if (!instance && !snapshots.isLoading) return <></>

  if (snapshots?.error) return <ErrorStub message={snapshots?.error} />

  return (
    <div className={classes.marginTop}>
      {snapshots.isLoading || isLoadingBranches ? (
        <Spinner size="lg" className={classes.spinner} />
      ) : (
        <>
          <SectionTitle
            level={2}
            tag="h2"
            text={`Snapshots (${filteredSnapshots?.length || 0})`}
            className={classes.sectionTitle}
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
                    <div style={{ display: 'flex' }}>
                      <InfoIcon className={classes.infoIcon} />
                    </div>
                  </Tooltip>
                )}
              </>
            }
          />
          <SnapshotHeader
            branches={branches}
            selectedBranch={selectedBranch || 'main'}
            setMessageFilter={setMessageFilter}
            setSelectedBranch={setSelectedBranch}
          />
          {!isEmpty ? (
            <SnapshotsList filteredSnapshots={filteredSnapshots} />
          ) : (
            <p className={classes.marginTop}>
              {messageFilter.length ? (
                <span>
                  No active snapshots found. Try removing the filter and
                  checking again
                </span>
              ) : (
                <span> This instance has no active snapshots</span>
              )}
            </p>
          )}
        </>
      )}
    </div>
  )
})
