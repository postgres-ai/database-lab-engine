import { TextField, Typography } from '@material-ui/core'
import Box from '@mui/material/Box'

import { FormValues, PhysicalEnv } from '../../useForm'
import { EnvsEditor } from '../EnvsEditor'

type Props = {
  values: FormValues
  onChange: <K extends keyof FormValues>(key: K, value: FormValues[K]) => void
  disabled?: boolean
  envsKeyErrors?: (string | undefined)[]
}

// WAL-G has one structured field (BackupName, defaulting to "LATEST"); storage
// backend, bucket, prefix, and credentials all live in the envs map. See
// engine/internal/retrieval/engine/postgres/physical/wal_g.go:36-38 and the
// example envs in config.example.physical_walg.yml:84-86.
const WALG_ENV_SUGGESTIONS = [
  'WALG_GS_PREFIX',
  'WALG_S3_PREFIX',
  'WALG_FILE_PREFIX',
  'GOOGLE_APPLICATION_CREDENTIALS',
  'AWS_ACCESS_KEY_ID',
  'AWS_SECRET_ACCESS_KEY',
  'AWS_REGION',
]

export const Walg = ({ values, onChange, disabled, envsKeyErrors }: Props) => {
  const onEnvsChange = (envs: PhysicalEnv[]) => onChange('physicalEnvs', envs)

  return (
    <Box mt={2} data-testid="walg-form">
      <Typography variant="subtitle1">WAL-G</Typography>
      <Box mt={1}>
        <TextField
          fullWidth
          label="Backup name"
          placeholder="LATEST"
          value={values.physicalWalgBackupName}
          disabled={disabled}
          onChange={(e) => onChange('physicalWalgBackupName', e.target.value)}
          helperText='Which backup to restore. "LATEST" picks the most recent.'
          inputProps={{ 'data-testid': 'walg-backup-name' }}
        />
      </Box>

      <Box mt={2}>
        <Typography variant="caption" color="textSecondary">
          Storage backend, bucket, prefix, and credentials all live in the envs
          map. Do not paste credentials into the backup name field or any URL.
        </Typography>
      </Box>

      <EnvsEditor
        envs={values.physicalEnvs}
        onChange={onEnvsChange}
        suggestions={WALG_ENV_SUGGESTIONS}
        disabled={disabled}
        keyErrors={envsKeyErrors}
      />
    </Box>
  )
}
