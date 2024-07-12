/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import cn from 'classnames'
import { useHistory } from 'react-router'
import { observer } from 'mobx-react-lite'
import { useEffect } from 'react'
import { TextField, makeStyles } from '@material-ui/core'

import { Button } from '@postgres.ai/shared/components/Button'
import { ResponseMessage } from '@postgres.ai/shared/pages/Instance/Configuration/ResponseMessage'
import { Select } from '@postgres.ai/shared/components/Select'
import { CreateBranchFormValues } from '@postgres.ai/shared/types/api/endpoints/createBranch'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinnerFlex'
import { Spinner } from '@postgres.ai/shared/components/Spinner'

import { useForm } from './useForm'
import { MainStoreApi } from './stores/Main'
import { useCreatedStores } from './useCreatedStores'
import { getCliBranchListCommand, getCliCreateBranchCommand } from './utils'

interface CreateBranchProps {
  api: MainStoreApi
  elements: {
    breadcrumbs: React.ReactNode
  }
}

const useStyles = makeStyles(
  {
    wrapper: {
      display: 'flex',
      gap: '60px',
      maxWidth: '1200px',
      fontSize: '14px',
      marginTop: '20px',

      '@media (max-width: 1300px)': {
        flexDirection: 'column',
        gap: '20px',
      },
    },
    container: {
      maxWidth: '100%',
      flex: '1 1 0',
      minWidth: 0,

      '&  p,span': {
        fontSize: 14,
      },
    },
    snippetContainer: {
      flex: '1 1 0',
      minWidth: 0,
      boxShadow: 'rgba(0, 0, 0, 0.1) 0px 4px 12px',
      padding: '10px 20px 10px 20px',
      height: 'max-content',
      borderRadius: '4px',
    },
    marginBottom: {
      marginBottom: '8px',
    },
    marginBottom2x: {
      marginBottom: '16px',
    },
    marginTop: {
      marginTop: '8px',
    },
    form: {
      marginTop: '16px',
    },
    spinner: {
      marginLeft: '8px',
      color: '#fff',
    },
    snapshotOverflow: {
      width: '100%',
      wordWrap: 'break-word',
      whiteSpace: 'initial',
    },
  },
  { index: 1 },
)

export const CreateBranchPage = observer(
  ({ api, elements }: CreateBranchProps) => {
    const stores = useCreatedStores(api)
    const classes = useStyles()
    const history = useHistory()

    const {
      load,
      snapshotListError,
      getBranchesError,
      createBranch,
      createBranchError,
      isBranchesLoading,
      isCreatingBranch,
      branchesList,
      snapshotsList,
    } = stores.main

    const handleSubmit = async (values: CreateBranchFormValues) => {
      await createBranch(values).then((branch) => {
        if (branch && branch?.name) {
          history.push(`/instance/branches/${branch.name}`)
        }
      })
    }

    const [{ formik }] = useForm(handleSubmit)

    useEffect(() => {
      load(formik.values.baseBranch)
    }, [formik.values.baseBranch])

    useEffect(() => {
      if (snapshotsList?.length) {
        formik.setFieldValue('snapshotID', snapshotsList[0]?.id)
      }
    }, [snapshotsList])

    if (isBranchesLoading) {
      return <StubSpinner />
    }

    return (
      <>
        {elements.breadcrumbs}
        <div className={classes.wrapper}>
          <div className={classes.container}>
            <SectionTitle tag="h1" level={1} text="Create branch" />
            {(snapshotListError || getBranchesError) && (
              <div className={classes.marginTop}>
                <ErrorStub
                  message={
                    snapshotListError?.message || getBranchesError?.message
                  }
                />
              </div>
            )}
            <div className={classes.form}>
              <TextField
                label="Branch name"
                variant="outlined"
                required
                fullWidth
                size="small"
                InputLabelProps={{
                  shrink: true,
                }}
                value={formik.values.branchName}
                error={Boolean(formik.errors.branchName)}
                className={classes.marginBottom}
                onChange={(e) =>
                  formik.setFieldValue('branchName', e.target.value)
                }
              />
              <p className={cn(classes.marginTop, classes.marginBottom)}>
                Choose an existing branch. The new branch will initially point
                at the same snapshot as the parent branch but going further,
                their evolution paths will be independent - new snapshots can be
                created for both branches.
              </p>
              <Select
                fullWidth
                label="Parent branch"
                value={formik.values.baseBranch}
                disabled={!branchesList || formik.isSubmitting}
                onChange={(e) =>
                  formik.setFieldValue('baseBranch', e.target.value)
                }
                error={Boolean(formik.errors.baseBranch)}
                items={
                  branchesList
                    ? branchesList.map((branch) => {
                        return {
                          value: branch.name,
                          children: branch.name,
                        }
                      })
                    : []
                }
              />
              <strong>Snapshot ID</strong>
              <p className={cn(classes.marginTop, classes.marginBottom)}>
                Choose an existing snapshot. This snapshot will be memorized as
                a forking point for the new branch; it cannot be deleted while
                the branch exists.
              </p>
              <Select
                fullWidth
                className={classes.marginBottom2x}
                label="Snapshot ID"
                value={formik.values.snapshotID}
                disabled={!branchesList || formik.isSubmitting}
                onChange={(e) =>
                  formik.setFieldValue('snapshotID', e.target.value)
                }
                error={Boolean(formik.errors.baseBranch)}
                items={
                  snapshotsList
                    ? snapshotsList.map((snapshot, i) => {
                        const isLatest = i === 0
                        return {
                          value: snapshot.id,
                          children: (
                            <div>
                              <strong className={classes.snapshotOverflow}>
                                {snapshot?.id} {isLatest && <span>Latest</span>}
                              </strong>
                              {snapshot?.dataStateAt && (
                                <p>Data state at: {snapshot?.dataStateAt}</p>
                              )}
                            </div>
                          ),
                        }
                      })
                    : []
                }
              />
              <Button
                variant="primary"
                size="medium"
                className={classes.marginTop}
                onClick={formik.submitForm}
              >
                Create branch
                {isCreatingBranch && (
                  <Spinner size="sm" className={classes.spinner} />
                )}
              </Button>
              {createBranchError && (
                <ResponseMessage
                  type={'error'}
                  message={createBranchError?.message}
                />
              )}
            </div>
          </div>{' '}
          <div className={classes.snippetContainer}>
            <SectionTitle tag="h1" level={1} text="The same using CLI" />
            <p className={classes.marginTop}>
              Alternatively, you can create a new branch using CLI. Fill the
              form, copy the command below and paste it into your terminal.
            </p>
            <SyntaxHighlight
              content={getCliCreateBranchCommand(formik.values.branchName, formik.values.baseBranch)}
            />
            <SectionTitle
              className={classes.marginTop}
              tag="h2"
              level={2}
              text={'Get branches using CLI'}
            />
            <p className={classes.marginTop}>
              You can get a list of all branches using CLI. Copy the command
              below and paste it into your terminal.
            </p>
            <SyntaxHighlight content={getCliBranchListCommand()} />
          </div>
        </div>
      </>
    )
  },
)
