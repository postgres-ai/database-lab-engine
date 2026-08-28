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

// The upgrade image ships binaries for the four majors preceding its own, which is the same
// bound the engine enforces on the request.
const MAX_MAJOR_JUMP = 4

// The oldest source the upgrade image can convert from is 12, and the target has to be newer than
// the source, so nothing below this can ever be a valid target. It is the one check that holds
// even when the clone's current version is unknown.
const MIN_TARGET_MAJOR = 13

type Props = {
  isOpen: boolean
  onClose: () => void
  clone: Clone
  onUpgradeClone: (targetVersion: number, dockerImage?: string) => void
}

const useStyles = makeStyles(
  {
    field: {
      margin: '16px 0 0 0',
    },
    hint: {
      margin: '8px 0 0 0',
    },
  },
  { index: 1 },
)

export const UpgradeCloneModal = (props: Props) => {
  const { isOpen, onClose, clone, onUpgradeClone } = props

  const classes = useStyles()

  const [targetVersion, setTargetVersion] = useState('')
  const [dockerImage, setDockerImage] = useState('')

  // The modal stays mounted while the clone page is open, so without this it reopens holding the
  // previous attempt's input - after a successful 16 to 17 upgrade it would show "17" already
  // typed, already rejected and already disabled, and a stale image would ride along unnoticed.
  useEffect(() => {
    if (isOpen) {
      setTargetVersion('')
      setDockerImage('')
    }
  }, [isOpen])

  const currentVersion = clone.dbVersion ? parseInt(clone.dbVersion, 10) : NaN
  const parsedTarget = parseInt(targetVersion, 10)

  const getValidationError = () => {
    if (!targetVersion) return null
    if (!/^\d+$/.test(targetVersion.trim())) {
      return 'Enter a PostgreSQL major version.'
    }

    if (parsedTarget < MIN_TARGET_MAJOR) {
      return `Target version must be ${MIN_TARGET_MAJOR} or newer.`
    }

    // Nothing below is decidable without the current version, so the engine has the last word.
    if (Number.isNaN(currentVersion)) return null

    if (parsedTarget <= currentVersion) {
      return `Target version must be greater than the current version (${currentVersion}).`
    }

    if (parsedTarget - currentVersion > MAX_MAJOR_JUMP) {
      return `The upgrade image covers at most ${MAX_MAJOR_JUMP} preceding majors.`
    }

    return null
  }

  const targetHint = clone.dbVersion
    ? 'For example: 17'
    : 'This clone has no recorded version, so the engine checks the target when the upgrade starts.'

  const validationError = getValidationError()
  const isUpgradeDisabled = !targetVersion || Boolean(validationError)

  const handleClickUpgrade = () => {
    if (isUpgradeDisabled) return

    onUpgradeClone(parsedTarget, dockerImage || undefined)
    onClose()
  }

  return (
    <Modal
      title={`Upgrade clone ${clone.id}`}
      isOpen={isOpen}
      onClose={onClose}
    >
      <Text>
        Clone <ImportantText>{clone.id}</ImportantText>
        {clone.dbVersion
          ? ` currently runs PostgreSQL ${clone.dbVersion}. `
          : ' has no recorded PostgreSQL version. '}
        Its data directory is upgraded in place, and the clone restarts on the
        target version. Resetting the clone afterwards returns it to the version
        the instance is configured with.
      </Text>
      <div className={classes.hint}>
        <Text>
          If the upgrade fails before any data is converted, the clone keeps
          running on its current version. If it fails afterwards, the clone is
          restored from its snapshot and changes made since it was created are
          lost.
        </Text>
      </div>
      <TextField
        label="Target major version"
        value={targetVersion}
        onChange={(e) => setTargetVersion(e.target.value)}
        error={Boolean(validationError)}
        helperText={validationError ?? targetHint}
        fullWidth
        className={classes.field}
      />
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
            isDisabled: isUpgradeDisabled,
          },
        ]}
      />
    </Modal>
  )
}
