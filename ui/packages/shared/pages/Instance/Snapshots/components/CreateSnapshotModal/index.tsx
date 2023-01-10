/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useHistory } from 'react-router'
import { useEffect, useState } from 'react'
import { TextField } from '@material-ui/core'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { Button } from '@postgres.ai/shared/components/Button'
import { ResponseMessage } from '@postgres.ai/shared/pages/Instance/Configuration/ResponseMessage'
import { Select } from '@postgres.ai/shared/components/Select'
import { MainStore } from '@postgres.ai/shared/pages/Instance/stores/Main'
import { ModalProps } from '@postgres.ai/shared/pages/Branches/components/Modals/types'
import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { generateSnapshotPageId } from '@postgres.ai/shared/pages/Instance/Snapshots/utils'

import { FormValues, useForm } from './useForm'

import styles from '../styles.module.scss'

interface CreateSnapshotModalProps extends ModalProps {
  createSnapshotError:
    | string
    | null
    | {
        title?: string
        message: string
      }
  createSnapshot: MainStore['createSnapshot']
  clones: Clone[]
  currentClone?: string
}

export const CreateSnapshotModal = ({
  isOpen,
  onClose,
  createSnapshotError,
  createSnapshot,
  clones,
  currentClone,
}: CreateSnapshotModalProps) => {
  const history = useHistory()
  const [snapshotError, setSnapshotError] = useState(createSnapshotError)

  const handleClose = () => {
    formik.resetForm()
    setSnapshotError('')
    onClose()
  }

  const handleSubmit = async (values: FormValues) => {
    await createSnapshot(values.cloneID, values.comment).then((snapshot) => {
      if (snapshot && generateSnapshotPageId(snapshot.snapshotID)) {
        history.push(
          `/instance/snapshots/${generateSnapshotPageId(snapshot.snapshotID)}`,
        )
      }
    })
  }

  useEffect(() => {
    setSnapshotError(createSnapshotError)
  }, [createSnapshotError])

  useEffect(() => {
    if (currentClone) formik.setFieldValue('cloneID', currentClone)
  }, [currentClone, isOpen])

  const [{ formik }] = useForm(handleSubmit)

  return (
    <Modal
      title="Create new snapshot"
      onClose={handleClose}
      isOpen={isOpen}
      size="sm"
    >
      <div className={styles.modalInputContainer}>
        <strong>Clone ID</strong>
        <p>
          Choose a clone ID from the dropdown below. This will be the starting
          point for your new snapshot.
        </p>
        <Select
          fullWidth
          label="Clone ID *"
          value={formik.values.cloneID}
          disabled={!clones || formik.isSubmitting}
          onChange={(e) => formik.setFieldValue('cloneID', e.target.value)}
          error={Boolean(formik.errors.cloneID)}
          items={
            clones
              ? clones.map((clone, i) => {
                  const isLatest = i === 0
                  const isCurrent = currentClone === clone?.id
                  return {
                    value: clone.id,
                    children: (
                      <div className={styles.selectContainer}>
                        <strong>
                          {clone.id} {isLatest && <span>Latest</span>}
                        </strong>
                        {isCurrent && (
                          <strong>
                            <span> Current</span>{' '}
                          </strong>
                        )}
                        <p>Created: {clone?.snapshot?.createdAt}</p>
                        <p>Data state at: {clone?.snapshot?.dataStateAt}</p>
                      </div>
                    ),
                  }
                })
              : []
          }
        />
        <strong>Comment</strong>
        <p className={styles.marginBottom}>
          Optional comment to be added to the snapshot.
        </p>
        <TextField
          label="Comment"
          variant="outlined"
          size="small"
          InputLabelProps={{
            shrink: true,
          }}
          value={formik.values.comment}
          error={Boolean(formik.errors.comment)}
          onChange={(e) => formik.setFieldValue('comment', e.target.value)}
        />
        <br />
        <Button
          variant="primary"
          size="medium"
          onClick={formik.submitForm}
          isDisabled={Boolean(formik.isSubmitting || !formik.values.cloneID)}
        >
          Create snapshot
        </Button>
        {snapshotError && (
          <ResponseMessage
            type={'error'}
            message={
              typeof snapshotError === 'string'
                ? snapshotError
                : snapshotError.message
            }
          />
        )}
      </div>
    </Modal>
  )
}
