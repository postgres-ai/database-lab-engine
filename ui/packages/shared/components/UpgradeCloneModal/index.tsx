/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useEffect, useState } from 'react'
import { makeStyles, TextField } from '@material-ui/core'
import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { Text } from '@postgres.ai/shared/components/Text'
import { Modal } from '@postgres.ai/shared/components/Modal'
import { ImportantText } from '@postgres.ai/shared/components/ImportantText'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'
import { colors } from '@postgres.ai/shared/styles/vars'

// The upgrade image ships binaries for the four majors preceding its own, which is the same
// bound the engine enforces on the request.
const MAX_MAJOR_JUMP = 4

type Props = {
  isOpen: boolean
  onClose: () => void
  clone: Clone
  // The version the instance upgrades to, read from its status. It follows from the configured
  // upgrade image, so it is shown rather than asked for.
  targetVersion: number
  onUpgradeClone: (dockerImage?: string) => void
}

const useStyles = makeStyles(
  {
    field: {
      margin: '16px 0 0 0',
    },
    hint: {
      margin: '8px 0 0 0',
    },
    error: {
      margin: '8px 0 0 0',
      color: colors.status.error,
    },
  },
  { index: 1 },
)

export const UpgradeCloneModal = (props: Props) => {
  const { isOpen, onClose, clone, targetVersion, onUpgradeClone } = props

  const classes = useStyles()

  const [dockerImage, setDockerImage] = useState('')

  // The modal stays mounted while the clone page is open, so without this it reopens holding the
  // previous attempt's image, which would ride along unnoticed.
  useEffect(() => {
    if (isOpen) setDockerImage('')
  }, [isOpen])

  const currentVersion = clone.dbVersion ? parseInt(clone.dbVersion, 10) : NaN

  // Nothing here is decidable without the current version, so the engine has the last word: it
  // reads the major out of the data directory once the upgrade starts.
  const getBlockingReason = () => {
    if (Number.isNaN(currentVersion)) return null

    if (targetVersion <= currentVersion) {
      return `This clone already runs PostgreSQL ${currentVersion}, and the instance upgrades to ${targetVersion}.`
    }

    if (targetVersion - currentVersion > MAX_MAJOR_JUMP) {
      return `The upgrade image covers at most ${MAX_MAJOR_JUMP} preceding majors, and this clone runs PostgreSQL ${currentVersion}.`
    }

    return null
  }

  const blockingReason = getBlockingReason()

  const handleClickUpgrade = () => {
    if (blockingReason) return

    onUpgradeClone(dockerImage || undefined)
    onClose()
  }

  return (
    <Modal title={`Upgrade clone ${clone.id}`} isOpen={isOpen} onClose={onClose}>
      <Text>
        Clone <ImportantText>{clone.id}</ImportantText>
        {clone.dbVersion
          ? ` currently runs PostgreSQL ${clone.dbVersion}. `
          : ' has no recorded PostgreSQL version. '}
        This instance upgrades clones to{' '}
        <ImportantText>PostgreSQL {targetVersion}</ImportantText>, the major its
        configured upgrade image is built on.
      </Text>
      {blockingReason ? (
        <div className={classes.error}>
          <Text>{blockingReason}</Text>
        </div>
      ) : (
        <>
          <div className={classes.hint}>
            <Text>
              The data directory is upgraded in place, and the clone restarts on
              the target version. Resetting the clone afterwards returns it to
              the version the instance is configured with.
            </Text>
          </div>
          <div className={classes.hint}>
            <Text>
              If the upgrade fails before any data is converted, the clone keeps
              running on its current version. If it fails afterwards, the clone
              is restored from its snapshot and changes made since it was
              created are lost.
            </Text>
          </div>
        </>
      )}
      <TextField
        label="Docker image (optional)"
        value={dockerImage}
        onChange={(e) => setDockerImage(e.target.value)}
        helperText="Leave empty to keep the current image and change only its major version"
        fullWidth
        className={classes.field}
      />
      <SimpleModalControls
        items={[
          {
            text: 'Cancel',
            onClick: onClose,
          },
          {
            text: 'Upgrade',
            onClick: handleClickUpgrade,
            variant: 'primary',
            isDisabled: Boolean(blockingReason),
          },
        ]}
      />
    </Modal>
  )
}
