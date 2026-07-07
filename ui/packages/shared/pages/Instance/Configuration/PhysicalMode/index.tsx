import {
  FormControl,
  FormControlLabel,
  Radio,
  RadioGroup,
  Typography,
} from '@material-ui/core'
import Box from '@mui/material/Box'

import { FormValues, PhysicalTool } from '../useForm'
import { PgBackRest } from './PgBackRest'
import { Sync } from './Sync'
import { Walg } from './Walg'

type Props = {
  values: FormValues
  onChange: <K extends keyof FormValues>(key: K, value: FormValues[K]) => void
  disabled?: boolean
  envsKeyErrors?: (string | undefined)[]
}

// Physical-mode restore tool selector + sub-form. Two values surface in the
// UI: WAL-G and pgBackRest, the two tool values the engine accepts outside
// customTool. pg_basebackup is invoked via the customTool path and remains
// YAML-only; when the loaded projection has `tool: customTool` we render a
// banner instead of the sub-form, keeping the user from accidentally wiping a
// hand-edited customTool config.
export const PhysicalMode = ({
  values,
  onChange,
  disabled,
  envsKeyErrors,
}: Props) => {
  const tool = values.physicalTool
  const isCustomTool = tool === 'customTool'

  return (
    <Box mt={1} data-testid="physical-mode">
      <Typography variant="subtitle2">Restore tool</Typography>
      {isCustomTool ? (
        <Box mt={1} data-testid="physical-custom-tool-banner">
          <Typography variant="body2" color="textSecondary">
            This config uses a custom restore tool (e.g. pg_basebackup). The UI
            cannot edit customTool configurations — edit the YAML directly.
          </Typography>
        </Box>
      ) : (
        <FormControl>
          <RadioGroup
            row
            value={tool}
            onChange={(_, value) =>
              onChange('physicalTool', value as PhysicalTool)
            }
            aria-label="physical restore tool"
          >
            <FormControlLabel
              value="walg"
              control={<Radio disabled={disabled} />}
              label="WAL-G"
            />
            <FormControlLabel
              value="pgbackrest"
              control={<Radio disabled={disabled} />}
              label="pgBackRest"
            />
          </RadioGroup>
        </FormControl>
      )}

      {!isCustomTool && tool === 'walg' && (
        <Walg
          values={values}
          onChange={onChange}
          disabled={disabled}
          envsKeyErrors={envsKeyErrors}
        />
      )}
      {!isCustomTool && tool === 'pgbackrest' && (
        <PgBackRest
          values={values}
          onChange={onChange}
          disabled={disabled}
          envsKeyErrors={envsKeyErrors}
        />
      )}

      {!isCustomTool && tool && (
        <Sync values={values} onChange={onChange} disabled={disabled} />
      )}
    </Box>
  )
}
