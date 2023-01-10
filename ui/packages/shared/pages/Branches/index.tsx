/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { observer } from 'mobx-react-lite'
import { makeStyles } from '@material-ui/core'
import React, { useEffect, useState } from 'react'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { Button } from '@postgres.ai/shared/components/Button2'
import { GetBranchesResponseType } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinner'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { BranchesTable } from '@postgres.ai/shared/pages/Branches/components/BranchesTable'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { CreateBranchModal } from '@postgres.ai/shared/pages/Branches/components/Modals/CreateBranchModal'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { InfoIcon } from '@postgres.ai/shared/icons/Info'

const useStyles = makeStyles(
  {
    container: {
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

export const Branches = observer((): React.ReactElement => {
  const stores = useStores()
  const classes = useStyles()
  const [branchesList, setBranchesList] = useState<GetBranchesResponseType[]>(
    [],
  )
  const [isCreateBranchOpen, setIsCreateBranchOpen] = useState(false)

  const {
    instance,
    getBranches,
    getSnapshotList,
    snapshotListError,
    isBranchesLoading,
    getBranchesError,
    createBranch,
    createBranchError,
  } = stores.main

  useEffect(() => {
    getBranches().then((response) => {
      response && setBranchesList(response)
    })
  }, [])

  if (!instance && !isBranchesLoading) return <></>

  if (getBranchesError)
    return (
      <ErrorStub
        title={getBranchesError?.title}
        message={getBranchesError?.message}
      />
    )

  if (isBranchesLoading) return <StubSpinner />

  return (
    <div className={classes.container}>
      <SectionTitle
        level={2}
        tag="h2"
        text={`Branches (${branchesList?.length || 0})`}
        rightContent={
          <>
            <Button
              theme="primary"
              isDisabled={!branchesList.length}
              onClick={() => setIsCreateBranchOpen(true)}
            >
              Create branch
            </Button>

            {!branchesList.length && (
              <Tooltip content="No existing branch">
                <InfoIcon className={classes.infoIcon} />
              </Tooltip>
            )}
          </>
        }
      />
      <BranchesTable
        branchesData={branchesList}
        emptyTableText="This instance has no active branches"
      />
      <CreateBranchModal
        isOpen={isCreateBranchOpen}
        onClose={() => setIsCreateBranchOpen(false)}
        createBranch={createBranch}
        createBranchError={createBranchError}
        branchesList={branchesList}
        getSnapshotList={getSnapshotList}
        snapshotListError={snapshotListError}
      />
    </div>
  )
})
