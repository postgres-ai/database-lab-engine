/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useEffect } from 'react'
import { useHistory } from 'react-router'
import { observer } from 'mobx-react-lite'
import { makeStyles } from '@material-ui/core'

import { Button } from '@postgres.ai/shared/components/Button'
import { ResponseMessage } from '@postgres.ai/shared/pages/Instance/Configuration/ResponseMessage'
import { Select } from '@postgres.ai/shared/components/Select'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinnerFlex'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { generateSnapshotPageId } from '@postgres.ai/shared/pages/Instance/Snapshots/utils'
import { TextField } from '@postgres.ai/shared/components/TextField'

import { FormValues, useForm } from './useForm'
import { MainStoreApi } from './stores/Main'
import { useCreatedStores } from './useCreatedStores'
import { getCliCreateSnapshotCommand } from './utils'

interface CreateSnapshotProps {
  instanceId: string
  api: MainStoreApi
  routes: {
    snapshot: (snapshotId: string) => string
  }
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
    marginTop2x: {
      marginTop: '16px',
    },
    spinner: {
      marginLeft: '8px',
      color: '#fff',
    },
  },
  { index: 1 },
)

export const CreateSnapshotPage = observer(
  ({ instanceId, api, elements, routes }: CreateSnapshotProps) => {
    const stores = useCreatedStores(api)
    const classes = useStyles()
    const history = useHistory()

    const {
      load,
      instance,
      createSnapshot,
      isCreatingSnapshot,
      snapshotError,
    } = stores.main

    const clonesList = instance?.instance?.state?.cloning.clones || []

    const handleSubmit = async (values: FormValues) => {
      await createSnapshot(values.cloneID, values.message, instanceId).then(
        (snapshot) => {
          if (snapshot && generateSnapshotPageId(snapshot.snapshotID)) {
            history.push(
              routes.snapshot(
                generateSnapshotPageId(snapshot.snapshotID) as string,
              ),
            )
          }
        },
      )
    }

    const [{ formik }] = useForm(handleSubmit)

    if (!clonesList) {
      return <StubSpinner />
    }

    useEffect(() => {
      load(instanceId)
    }, [])

    useEffect(() => {
      if (!history.location.search) return

      const queryString = history.location.search.split('?')[1]

      if (!queryString) return

      const params = new URLSearchParams(queryString)

      const cloneID = params.get('clone_id')

      if (!cloneID) return

      formik.setFieldValue('cloneID', cloneID)

    }, [history.location.search, formik.initialValues])

    return (
      <>
        {elements.breadcrumbs}
        <div className={classes.wrapper}>
          <div className={classes.container}>
            <SectionTitle tag="h1" level={1} text="Create Snapshot" />
            <div className={classes.marginTop2x}>
              <strong>Clone ID</strong>
              <p className={classes.marginTop}>
                Choose a clone ID from the dropdown below. This will be the
                starting point for your new snapshot.
              </p>
              <Select
                fullWidth
                label="Clone ID *"
                value={formik.values.cloneID}
                disabled={formik.isSubmitting}
                className={classes.marginBottom2x}
                onChange={(e) =>
                  formik.setFieldValue('cloneID', e.target.value)
                }
                error={Boolean(formik.errors.cloneID)}
                items={
                  clonesList
                    ? clonesList.map((clone, i) => {
                        const isLatest = i === 0
                        return {
                          value: clone.id,
                          children: (
                            <div>
                              <strong>
                                {clone.id} {isLatest && <span>Latest</span>}
                              </strong>
                              <p>Created: {clone?.snapshot?.createdAt}</p>
                              <p>
                                Data state at: {clone?.snapshot?.dataStateAt}
                              </p>
                            </div>
                          ),
                        }
                      })
                    : []
                }
              />
              <strong>Message</strong>
              <p className={classes.marginTop}>
                Message to be added to the snapshot.
              </p>
              <TextField
                label="Message *"
                fullWidth
                className={classes.marginBottom2x}
                value={formik.values.message}
                error={Boolean(formik.errors.message)}
                onChange={(e) =>
                  formik.setFieldValue('message', e.target.value)
                }
              />
              <Button
                variant="primary"
                size="medium"
                className={classes.marginTop}
                onClick={formik.submitForm}
              >
                Create snapshot
                {isCreatingSnapshot && (
                  <Spinner size="sm" className={classes.spinner} />
                )}
              </Button>
              {snapshotError && (
                <ResponseMessage type={'error'} message={snapshotError} />
              )}
            </div>
          </div>{' '}
          <div className={classes.snippetContainer}>
            <SectionTitle tag="h1" level={1} text="The same using CLI" />
            <p className={classes.marginTop}>
              Alternatively, you can create a new snapshot using CLI. Fill the
              form, copy the command below and paste it into your terminal.
            </p>
            <SyntaxHighlight
              content={getCliCreateSnapshotCommand(
                formik.values.cloneID,
                formik.values.message,
              )}
            />
          </div>
        </div>
      </>
    )
  },
)
