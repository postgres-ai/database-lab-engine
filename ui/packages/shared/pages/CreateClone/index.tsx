import { useEffect, useState } from 'react'
import { useHistory } from 'react-router-dom'
import { observer } from 'mobx-react-lite'
import { useTimer } from 'use-timer'
import { Paper, FormControlLabel, Checkbox } from '@material-ui/core'
import { Info as InfoIcon } from '@material-ui/icons'

import { StubSpinner } from '@postgres.ai/shared/components/StubSpinnerFlex'
import { TextField } from '@postgres.ai/shared/components/TextField'
import { Select } from '@postgres.ai/shared/components/Select'
import { Button } from '@postgres.ai/shared/components/Button'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { compareSnapshotsDesc } from '@postgres.ai/shared/utils/snapshot'
import { round } from '@postgres.ai/shared/utils/numbers'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'

import { useCreatedStores, MainStoreApi } from './useCreatedStores'
import { useForm, FormValues } from './useForm'

import styles from './styles.module.scss'

type Host = {
  instanceId: string
  routes: {
    clone: (cloneId: string) => string
  }
  api: MainStoreApi
  elements: {
    breadcrumbs: React.ReactNode
  }
}

type Props = Host

export const CreateClone = observer((props: Props) => {
  const history = useHistory()
  const stores = useCreatedStores(props.api)
  const timer = useTimer()
  const [branchesList, setBranchesList] = useState<string[]>([])

  // Form.
  const onSubmit = async (values: FormValues) => {
    timer.start()

    const isSuccess = await stores.main.createClone(values)

    if (!isSuccess) {
      timer.pause()
      timer.reset()
    }
  }

  const formik = useForm(onSubmit)

  // Initial loading data.
  useEffect(() => {
    stores.main.load(props.instanceId)

    stores.main.getBranches().then((response) => {
      if (response) {
        setBranchesList(response.map((branch) => branch.name))
      }
    })
  }, [])

  // Redirect when clone is created and stable.
  useEffect(() => {
    if (!stores.main.clone) return
    if (!stores.main.isCloneStable) return

    history.push(props.routes.clone(stores.main.clone.id))
  }, [stores.main.clone, stores.main.isCloneStable])

  // Snapshots.
  const sortedSnapshots = stores.main.snapshots.data
    ?.slice()
    .sort(compareSnapshotsDesc)

  useEffect(() => {
    const [firstSnapshot] = sortedSnapshots ?? []
    if (!firstSnapshot) return

    formik.setFieldValue('snapshotId', firstSnapshot.id)
  }, [Boolean(sortedSnapshots)])

  const headRendered = (
    <>
      {/* //TODO: make global reset styles. */}
      <style>{'p { margin: 0; }'}</style>

      {props.elements.breadcrumbs}
      <SectionTitle
        className={styles.title}
        tag="h1"
        level={1}
        text="Create clone"
      />
    </>
  )

  // Initial loading spinner.
  if (!stores.main.instance || !stores.main.snapshots.data)
    return (
      <>
        {headRendered}

        <StubSpinner />
      </>
    )

  // Instance/branches getting error.
  if (stores.main.instanceError || stores.main.getBranchesError)
    return (
      <>
        {headRendered}

        <ErrorStub
          message={
            stores.main.instanceError || stores.main.getBranchesError?.message
          }
        />
      </>
    )

  // Snapshots getting error.
  if (stores.main.snapshots.error)
    return <ErrorStub {...stores.main.snapshots.error} />

  const isCloneUnstable = Boolean(
    stores.main.clone && !stores.main.isCloneStable,
  )
  const isCreatingClone = formik.isSubmitting || isCloneUnstable

  return (
    <>
      {headRendered}

      <div className={styles.form}>
        {stores.main.cloneError && (
          <div className={styles.section}>
            <ErrorStub message={stores.main.cloneError} />
          </div>
        )}

        <div className={styles.section}>
          {branchesList && branchesList.length > 0 && (
            <Select
              fullWidth
              label="Branch"
              value={formik.values.branch}
              disabled={!branchesList || isCreatingClone}
              onChange={(e) => formik.setFieldValue('branch', e.target.value)}
              error={Boolean(formik.errors.branch)}
              items={
                branchesList?.map((snapshot) => {
                  return {
                    value: snapshot,
                    children: snapshot,
                  }
                }) ?? []
              }
            />
          )}

          <TextField
            fullWidth
            label="Clone ID"
            value={formik.values.cloneId}
            onChange={(e) => formik.setFieldValue('cloneId', e.target.value)}
            error={Boolean(formik.errors.cloneId)}
            disabled={isCreatingClone}
          />

          <Select
            fullWidth
            label="Data state time *"
            value={formik.values.snapshotId}
            disabled={!sortedSnapshots || isCreatingClone}
            onChange={(e) => formik.setFieldValue('snapshotId', e.target.value)}
            error={Boolean(formik.errors.snapshotId)}
            items={
              sortedSnapshots?.map((snapshot, i) => {
                const isLatest = i === 0
                return {
                  value: snapshot.id,
                  children: (
                    <>
                      {snapshot.dataStateAt}
                      {isLatest && (
                        <span className={styles.snapshotTag}>Latest</span>
                      )}
                    </>
                  ),
                }
              }) ?? []
            }
          />

          <p className={styles.remark}>
            By default latest snapshot of database is used. You can select&nbsp;
            different snapshots if earlier database state is needed
          </p>
        </div>

        <div className={styles.section}>
          <h2 className={styles.title}>Database credentials *</h2>

          <p className={styles.text}>
            Set custom credentials for the new clone. Save the password in
            reliable place, it can’t be read later.
          </p>

          <TextField
            fullWidth
            label="Database username *"
            value={formik.values.dbUser}
            onChange={(e) => formik.setFieldValue('dbUser', e.target.value)}
            error={Boolean(formik.errors.dbUser)}
            disabled={isCreatingClone}
          />

          <TextField
            fullWidth
            label="Database password *"
            type="password"
            value={formik.values.dbPassword}
            onChange={(e) => formik.setFieldValue('dbPassword', e.target.value)}
            error={Boolean(formik.errors.dbPassword)}
            disabled={isCreatingClone}
          />
        </div>

        <div className={styles.section}>
          <Paper className={styles.summary}>
            <InfoIcon className={styles.summaryIcon} />

            <div className={styles.params}>
              <p className={styles.param}>
                <span>Data size:</span>
                <strong>
                  {stores.main.instance.state.dataSize
                    ? formatBytesIEC(stores.main.instance.state.dataSize)
                    : '-'}
                </strong>
              </p>

              <p className={styles.param}>
                <span>Expected cloning time:</span>
                <strong>
                  {round(
                    stores.main.instance.state.cloning.expectedCloningTime,
                    2,
                  )}{' '}
                  s
                </strong>
              </p>
            </div>
          </Paper>
        </div>

        <div className={styles.section}>
          <FormControlLabel
            label="Enable deletion protection"
            control={
              <Checkbox
                checked={formik.values.isProtected}
                onChange={(e) =>
                  formik.setFieldValue('isProtected', e.target.checked)
                }
                name="protected"
                disabled={isCreatingClone}
              />
            }
          />

          <p className={styles.remark}>
            When enabled no one can delete this clone and automated deletion is
            also disabled.
            <br />
            Please be careful: abandoned clones with this checkbox enabled may
            cause out-of-disk-space events. Check disk space on daily basis and
            delete this clone once the work is done.
          </p>
        </div>

        <div className={styles.section}>
          <div className={styles.controls}>
            <Button
              onClick={formik.submitForm}
              variant="primary"
              size="medium"
              isDisabled={isCreatingClone}
            >
              Create clone
              {isCreatingClone && (
                <Spinner size="sm" className={styles.spinner} />
              )}
            </Button>

            {isCreatingClone && (
              <p className={styles.elapsedTime}>Elapsed time: {timer.time} s</p>
            )}
          </div>
        </div>
      </div>
    </>
  )
})
