import {
  Checkbox,
  FormControlLabel,
  TextField,
  Typography,
} from '@material-ui/core'
import Box from '@mui/material/Box'

import { FormValues, PhysicalEnv } from '../../useForm'
import { EnvsEditor } from '../EnvsEditor'

type Props = {
  values: FormValues
  onChange: <K extends keyof FormValues>(key: K, value: FormValues[K]) => void
  disabled?: boolean
  envsKeyErrors?: (string | undefined)[]
}

// pgBackRest exposes two structured fields (stanza, delta); repo paths, S3
// keys, archive options all live in the envs map. See
// engine/internal/retrieval/engine/postgres/physical/pgbackrest.go:23-26 and
// the example envs in config.example.physical_pgbackrest.yml:84-99.
const PGBACKREST_ENV_SUGGESTIONS = [
  'PGBACKREST_REPO',
  'PGBACKREST_REPO1_TYPE',
  'PGBACKREST_REPO1_PATH',
  'PGBACKREST_REPO1_HOST',
  'PGBACKREST_REPO1_HOST_USER',
  'PGBACKREST_REPO1_S3_BUCKET',
  'PGBACKREST_REPO1_S3_ENDPOINT',
  'PGBACKREST_REPO1_S3_KEY',
  'PGBACKREST_REPO1_S3_KEY_SECRET',
  'PGBACKREST_REPO1_S3_REGION',
  'PGBACKREST_LOG_LEVEL_CONSOLE',
  'PGBACKREST_PROCESS_MAX',
]

export const PgBackRest = ({
  values,
  onChange,
  disabled,
  envsKeyErrors,
}: Props) => {
  const onEnvsChange = (envs: PhysicalEnv[]) => onChange('physicalEnvs', envs)

  return (
    <Box mt={2} data-testid="pgbackrest-form">
      <Typography variant="subtitle1">pgBackRest</Typography>
      <Box mt={1}>
        <TextField
          fullWidth
          label="Stanza"
          placeholder="my-stanza"
          value={values.physicalPgbackrestStanza}
          disabled={disabled}
          onChange={(e) => onChange('physicalPgbackrestStanza', e.target.value)}
          helperText="Stanza name (must match the stanza configured in your pgBackRest setup)."
          inputProps={{ 'data-testid': 'pgbackrest-stanza' }}
        />
      </Box>
      <Box mt={1}>
        <FormControlLabel
          control={
            <Checkbox
              checked={values.physicalPgbackrestDelta}
              disabled={disabled}
              onChange={(e) =>
                onChange('physicalPgbackrestDelta', e.target.checked)
              }
              inputProps={{
                'aria-label': 'delta',
                'data-testid': 'pgbackrest-delta',
              } as React.InputHTMLAttributes<HTMLInputElement>}
            />
          }
          label="Delta restore"
        />
      </Box>

      <EnvsEditor
        envs={values.physicalEnvs}
        onChange={onEnvsChange}
        suggestions={PGBACKREST_ENV_SUGGESTIONS}
        disabled={disabled}
        keyErrors={envsKeyErrors}
      />
    </Box>
  )
}
