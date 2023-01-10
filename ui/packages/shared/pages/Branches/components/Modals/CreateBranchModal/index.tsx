/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useHistory } from 'react-router'
import { useState, useEffect } from 'react'
import { TextField, makeStyles } from '@material-ui/core'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { Button } from '@postgres.ai/shared/components/Button'
import { ResponseMessage } from '@postgres.ai/shared/pages/Instance/Configuration/ResponseMessage'
import { Select } from '@postgres.ai/shared/components/Select'
import { MainStore } from '@postgres.ai/shared/pages/Instance/stores/Main'
import { ModalProps } from '@postgres.ai/shared/pages/Branches/components/Modals/types'
import { CreateBranchFormValues } from '@postgres.ai/shared/types/api/endpoints/createBranch'
import { GetBranchesResponseType } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import { GetSnapshotListResponseType } from '@postgres.ai/shared/types/api/endpoints/getSnapshotList'

import { useForm } from './useForm'

import styles from '../styles.module.scss'

interface CreateBranchModalProps extends ModalProps {
  createBranchError: string | null
  snapshotListError: string | null
  createBranch: MainStore['createBranch']
  branchesList: GetBranchesResponseType[] | null
  getSnapshotList: MainStore['getSnapshotList']
}

const useStyles = makeStyles(
  {
    marginBottom: {
      marginBottom: '8px',
    },
    marginTop: {
      marginTop: '8px',
    },
  },
  { index: 1 },
)

export const CreateBranchModal = ({
  isOpen,
  onClose,
  createBranchError,
  snapshotListError,
  createBranch,
  branchesList,
  getSnapshotList,
}: CreateBranchModalProps) => {
  const classes = useStyles()
  const history = useHistory()
  const [branchError, setBranchError] = useState(createBranchError)
  const [snapshotsList, setSnapshotsList] = useState<
    GetSnapshotListResponseType[] | null
  >()

  const handleClose = () => {
    formik.resetForm()
    setBranchError('')
    onClose()
  }

  const handleSubmit = async (values: CreateBranchFormValues) => {
    await createBranch(values).then((branch) => {
      if (branch && branch?.name) {
        history.push(`/instance/branches/${branch.name}`)
      }
    })
  }

  const [{ formik, isFormDisabled }] = useForm(handleSubmit)

  useEffect(() => {
    setBranchError(createBranchError || snapshotListError)
  }, [createBranchError, snapshotListError])

  useEffect(() => {
    if (isOpen) {
      getSnapshotList(formik.values.baseBranch).then((res) => {
        if (res) {
          const filteredSnapshots = res.filter((snapshot) => snapshot.id)
          setSnapshotsList(filteredSnapshots)
          formik.setFieldValue('snapshotID', filteredSnapshots[0]?.id)
        }
      })
    }
  }, [isOpen, formik.values.baseBranch])

  return (
    <Modal
      title="Create new branch"
      onClose={handleClose}
      isOpen={isOpen}
      size="sm"
    >
      <div className={styles.modalInputContainer}>
        <TextField
          label="Branch name"
          variant="outlined"
          required
          size="small"
          InputLabelProps={{
            shrink: true,
          }}
          value={formik.values.branchName}
          error={Boolean(formik.errors.branchName)}
          className={classes.marginBottom}
          onChange={(e) => formik.setFieldValue('branchName', e.target.value)}
        />
        <strong>Parent branch</strong>
        <p>
          Choose an existing branch. The new branch will initially point at the
          same snapshot as the parent branch but going further, their evolution
          paths will be independent - new snapshots can be created for both
          branches.
        </p>
        <Select
          fullWidth
          label="Parent branch"
          value={formik.values.baseBranch}
          disabled={!branchesList || formik.isSubmitting}
          onChange={(e) => formik.setFieldValue('baseBranch', e.target.value)}
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
        <p>
          Choose an existing snapshot. This snapshot will be memorized as a
          forking point for the new branch; it cannot be deleted while the
          branch exists.
        </p>
        <Select
          fullWidth
          label="Snapshot ID"
          value={formik.values.snapshotID}
          disabled={!branchesList || formik.isSubmitting}
          onChange={(e) => formik.setFieldValue('snapshotID', e.target.value)}
          error={Boolean(formik.errors.baseBranch)}
          items={
            snapshotsList
              ? snapshotsList.map((snapshot, i) => {
                  const isLatest = i === 0
                  return {
                    value: snapshot.id,
                    children: (
                      <div className={styles.selectContainer}>
                        <strong className={styles.snapshotOverflow}>
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
          isDisabled={isFormDisabled}
        >
          Create branch
        </Button>
        {branchError ||
          (snapshotListError && (
            <ResponseMessage
              type={'error'}
              message={branchError || snapshotListError}
            />
          ))}
      </div>
    </Modal>
  )
}
