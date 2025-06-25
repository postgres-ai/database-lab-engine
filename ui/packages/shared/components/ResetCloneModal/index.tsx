/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useEffect, useState } from 'react'
import { makeStyles } from '@material-ui/core'
import { formatDistanceToNowStrict } from 'date-fns'

import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'
import { Text } from '@postgres.ai/shared/components/Text'
import { Modal } from '@postgres.ai/shared/components/Modal'
import { Select } from '@postgres.ai/shared/components/Select'
import { ImportantText } from '@postgres.ai/shared/components/ImportantText'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'
import { compareSnapshotsDesc } from '@postgres.ai/shared/utils/snapshot'
import { isValidDate } from '@postgres.ai/shared/utils/date'

type Props = {
  isOpen: boolean
  onClose: () => void
  clone: Clone
  onResetClone: (snapshotId: string) => void
  snapshots: Snapshot[] | null
  version: string | null | undefined
}

const useStyles = makeStyles(
  {
    snapshots: {
      margin: '16px 0 0 0',
    },
    snapshotTag: {
      marginLeft: '4px',
      fontWeight: 700,
    },
  },
  { index: 1 },
)

export const ResetCloneModal = (props: Props) => {
  const { isOpen, onClose, clone, onResetClone, snapshots } = props

  const classes = useStyles()

  // `slice` here is the mobx requirement.
  const sortedSnapshots = snapshots?.slice().sort(compareSnapshotsDesc)

  const [selectedSnapshotId, setSelectedSnapshotId] = useState<string | null>(
    null,
  )

  const selectCurrentSnapshot = () => {
    if (!sortedSnapshots) return

    const currentSnapshot = sortedSnapshots.find(
      (snapshot) => snapshot.id === clone.snapshot?.id,
    )

    if (!currentSnapshot) return

    setSelectedSnapshotId(currentSnapshot.id)
  }

  useEffect(selectCurrentSnapshot, [Boolean(sortedSnapshots)])

  const handleClickReset = () => {
    if (!selectedSnapshotId) return
    onResetClone(selectedSnapshotId)
    onClose()
  }

  const handleClickResetToLatest = () => {
    if (!sortedSnapshots) return
    const [latestSnapshot] = sortedSnapshots
    if (!latestSnapshot) return
    onResetClone(latestSnapshot.id)
    onClose()
  }

  const isSnapshotsSelectingSupported = Boolean(props.version)

  const isSnapshotsSelectingDisabled =
    !sortedSnapshots || !isSnapshotsSelectingSupported

  return (
    <Modal
      title={`Reset clone ${clone.id}`}
      isOpen={isOpen}
      onClose={onClose}
      titleRightContent={!sortedSnapshots && <Spinner size="sm" />}
    >
      <Text>
        All changes to the clone <ImportantText>{clone.id}</ImportantText> will
        be reset to chosen data state time.
      </Text>
      <Select
        disabled={isSnapshotsSelectingDisabled}
        label="Data state time"
        value={selectedSnapshotId}
        items={
          sortedSnapshots?.map((snapshot, i) => {
            const isLatest = i === 0
            const isCurrent = snapshot.id === clone.snapshot?.id

            return {
              value: snapshot.id,
              children: (
                <>
                  {snapshot.dataStateAt} (
                  {isValidDate(snapshot.dataStateAtDate) && 
                    formatDistanceToNowStrict(snapshot.dataStateAtDate, {
                      addSuffix: true,
                    })}
                  )
                  {isLatest && (
                    <span className={classes.snapshotTag}>Latest</span>
                  )}
                  {isCurrent && (
                    <span className={classes.snapshotTag}>Current</span>
                  )}
                </>
              ),
            }
          }) ?? []
        }
        onChange={(e) => setSelectedSnapshotId(e.target.value)}
        fullWidth={true}
        className={classes.snapshots}
      />
      <SimpleModalControls
        items={[
          {
            text: 'Cancel',
            onClick: onClose,
          },
          {
            text: 'Reset to latest',
            onClick: handleClickResetToLatest,
            isDisabled: isSnapshotsSelectingDisabled,
          },
          {
            text: 'Reset',
            isDisabled: !selectedSnapshotId,
            variant: 'primary',
            onClick: handleClickReset,
          },
        ]}
      />
    </Modal>
  )
}
