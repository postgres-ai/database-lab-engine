import {
  Checkbox,
  FormControlLabel,
  TextField,
  Typography,
} from '@material-ui/core'
import Box from '@mui/material/Box'

import { FormValues } from '../../useForm'

type Props = {
  values: FormValues
  onChange: <K extends keyof FormValues>(key: K, value: FormValues[K]) => void
  disabled?: boolean
}

// Shared section rendered below WAL-G / pgBackRest with the structured fields
// that apply across all physical sub-tools (sync.enabled and dockerImage). The
// engine surfaces more knobs (sync.healthCheck, sync.configs, recovery target,
// custom restore command), but they remain YAML-only for 4.2 to keep the
// projection footprint flat.
export const Sync = ({ values, onChange, disabled }: Props) => {
  return (
    <Box mt={2} data-testid="physical-sync">
      <Typography variant="subtitle1">Sync container & image</Typography>
      <Box mt={1}>
        <TextField
          fullWidth
          label="Docker image"
          placeholder="postgresai/extended-postgres:18-0.6.2"
          value={values.physicalDockerImage}
          disabled={disabled}
          onChange={(e) => onChange('physicalDockerImage', e.target.value)}
          helperText="Postgres image for restore/sync containers. Major version must match the source."
          inputProps={{ 'data-testid': 'physical-docker-image' }}
        />
      </Box>
      <Box mt={1}>
        <FormControlLabel
          control={
            <Checkbox
              checked={values.physicalSyncEnabled}
              disabled={disabled}
              onChange={(e) =>
                onChange('physicalSyncEnabled', e.target.checked)
              }
              inputProps={{
                'aria-label': 'sync enabled',
                'data-testid': 'physical-sync-enabled',
              } as React.InputHTMLAttributes<HTMLInputElement>}
            />
          }
          label="Run sync container"
        />
      </Box>
      <Box mt={1}>
        <Typography variant="caption" color="textSecondary">
          For advanced sync settings (health check, sync postgres configs,
          recovery target), edit the YAML config directly.
        </Typography>
      </Box>
    </Box>
  )
}
