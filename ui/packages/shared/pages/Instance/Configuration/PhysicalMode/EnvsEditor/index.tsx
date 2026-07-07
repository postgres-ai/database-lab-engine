import { Button, IconButton, TextField, Typography } from '@material-ui/core'
import Box from '@mui/material/Box'

import { PhysicalEnv } from '../../useForm'

type Props = {
  envs: PhysicalEnv[]
  onChange: (envs: PhysicalEnv[]) => void
  suggestions?: string[]
  disabled?: boolean
  keyErrors?: (string | undefined)[]
}

// EnvsEditor renders a rows-of-key/value editor with add/remove and a
// click-to-add suggestion list. Engine consumes envs as a free-form map
// (physical.go:76, CopyOptions.Envs map[string]string); the UI is a thin
// surface over that map.
export const EnvsEditor = ({
  envs,
  onChange,
  suggestions = [],
  disabled,
  keyErrors = [],
}: Props) => {
  const updateRow = (i: number, patch: Partial<PhysicalEnv>) => {
    const next = envs.slice()
    next[i] = { ...next[i], ...patch }
    onChange(next)
  }

  const removeRow = (i: number) => {
    const next = envs.slice()
    next.splice(i, 1)
    onChange(next)
  }

  const addRow = (key = '') => {
    onChange([...envs, { key, value: '' }])
  }

  const usedKeys = new Set(envs.map((e) => e.key))

  return (
    <Box mt={1} data-testid="envs-editor">
      <Typography variant="subtitle2">Environment variables</Typography>
      {envs.length === 0 ? (
        <Box mt={1} mb={1}>
          <Typography variant="caption" color="textSecondary">
            No environment variables set. Use suggestions below or click "Add".
          </Typography>
        </Box>
      ) : (
        envs.map((env, i) => (
          <Box
            key={i}
            display="flex"
            alignItems="center"
            mt={1}
            data-testid={`envs-row-${i}`}
          >
            <TextField
              size="small"
              label="Name"
              value={env.key}
              disabled={disabled}
              error={Boolean(keyErrors[i])}
              helperText={keyErrors[i]}
              onChange={(e) => updateRow(i, { key: e.target.value })}
              inputProps={{ 'data-testid': `envs-key-${i}` }}
            />
            <Box mx={1}>
              <TextField
                size="small"
                label="Value"
                value={env.value}
                disabled={disabled}
                onChange={(e) => updateRow(i, { value: e.target.value })}
                inputProps={{ 'data-testid': `envs-value-${i}` }}
              />
            </Box>
            <IconButton
              size="small"
              aria-label="remove env"
              disabled={disabled}
              onClick={() => removeRow(i)}
              data-testid={`envs-remove-${i}`}
            >
              ×
            </IconButton>
          </Box>
        ))
      )}

      <Box mt={1}>
        <Button
          size="small"
          variant="outlined"
          disabled={disabled}
          onClick={() => addRow()}
          data-testid="envs-add"
        >
          + Add
        </Button>
      </Box>

      {suggestions.length > 0 && (
        <Box mt={1}>
          <Typography variant="caption" color="textSecondary">
            Suggestions:
          </Typography>
          <Box mt={0.5}>
            {suggestions.map((s) => (
              <Button
                key={s}
                size="small"
                variant="text"
                disabled={disabled || usedKeys.has(s)}
                onClick={() => addRow(s)}
                data-testid={`envs-suggest-${s}`}
              >
                {s}
              </Button>
            ))}
          </Box>
        </Box>
      )}
    </Box>
  )
}
