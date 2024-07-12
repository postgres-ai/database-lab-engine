/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { observer } from 'mobx-react-lite'
import { useHistory } from 'react-router'
import { makeStyles } from '@material-ui/core'
import React, { useEffect, useState } from 'react'

import { useStores, useHost } from '@postgres.ai/shared/pages/Instance/context'
import { Button } from '@postgres.ai/shared/components/Button2'
import { GetBranchesResponseType } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { BranchesTable } from '@postgres.ai/shared/pages/Branches/components/BranchesTable'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
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
    spinner: {
      position: 'absolute',
      right: '50%',
      transform: 'translate(-50%, -50%)',
    },
  },
  { index: 1 },
)

export const Branches = observer((): React.ReactElement => {
  const host = useHost()
  const stores = useStores()
  const classes = useStyles()
  const history = useHistory()
  const [branchesList, setBranchesList] = useState<GetBranchesResponseType[]>(
    [],
  )
  const {
    instance,
    getBranches,
    isBranchesLoading,
    getBranchesError,
    deleteBranchError,
    deleteBranch,
  } = stores.main

  const goToBranchAddPage = () => history.push(host.routes.createBranch())

  const handleDestroyBranch = async (branchId: string) => {
    const isSuccess = await deleteBranch(branchId)
    if (isSuccess) history.push('/instance/branches')
  }

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

  return (
    <div className={classes.container}>
      {isBranchesLoading ? (
        <Spinner size="lg" className={classes.spinner} />
      ) : (
        <>
          <SectionTitle
            level={2}
            tag="h2"
            text={`Branches (${branchesList?.length || 0})`}
            rightContent={
              <>
                <Button
                  theme="primary"
                  isDisabled={!branchesList.length}
                  onClick={goToBranchAddPage}
                >
                  Create branch
                </Button>

                {!branchesList.length && (
                  <Tooltip content="No existing branch">
                    <div style={{ display: 'flex' }}>
                      <InfoIcon className={classes.infoIcon} />
                    </div>
                  </Tooltip>
                )}
              </>
            }
          />
          <BranchesTable
            branchesData={branchesList}
            deleteBranch={handleDestroyBranch}
            deleteBranchError={deleteBranchError}
            emptyTableText="This instance has no active branches"
          />
        </>
      )}
    </div>
  )
})
