import { Box, TextField, Chip } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { InfoIcon } from '@postgres.ai/shared/icons/Info'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { uniqueDatabases } from '../utils'

import styles from '../styles.module.scss'
import classNames from 'classnames'

const useStyles = makeStyles({
  textField: {
    '& .MuiOutlinedInput-notchedOutline': {
      borderColor: '#000 !important',
    },
  },
})

export const InputWithTooltip = ({
  value,
  label,
  error,
  onChange,
  tooltipText,
  disabled,
}: {
  value?: string
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  tooltipText: () => React.ReactNode
  label: string
  error?: string
  disabled: boolean
}) => {
  const classes = useStyles()

  return (
    <Box mt={2} mb={2} display="flex" alignItems="center">
      <TextField
        className={classNames(!disabled && classes.textField, styles.textField)}
        label={label}
        variant="outlined"
        size="small"
        value={value}
        error={Boolean(error)}
        onChange={onChange}
        disabled={disabled}
      />
      <Tooltip interactive content={<p>{tooltipText()}</p>}>
        <InfoIcon className={styles.infoIcon} />
      </Tooltip>
    </Box>
  )
}

export const InputWithChip = ({
  value,
  label,
  id,
  onChange,
  tooltipText,
  disabled,
  handleDeleteDatabase,
}: {
  value: string
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  tooltipText: () => React.ReactNode
  handleDeleteDatabase: (event: any, database: string) => void
  label: string
  id: string
  disabled: boolean
}) => {
  const classes = useStyles()

  return (
    <Box mt={2} mb={2}>
      <Box display="flex" alignItems="center">
        <TextField
          className={classNames(
            !disabled && classes.textField,
            styles.textField,
          )}
          variant="outlined"
          onChange={onChange}
          value={value}
          multiline
          disabled={disabled}
          label={label}
          inputProps={{
            name: id,
            id: id,
          }}
          InputLabelProps={{
            shrink: true,
          }}
        />
        <Tooltip interactive content={<p>{tooltipText()}</p>}>
          <InfoIcon className={styles.infoIcon} />
        </Tooltip>
      </Box>
      <div>
        {value &&
          uniqueDatabases(value)
            .split(' ')
            .map((database, index) => {
              if (database !== '') {
                return (
                  <Chip
                    key={index}
                    className={styles.chip}
                    label={database}
                    onDelete={(event) => handleDeleteDatabase(event, database)}
                    color="primary"
                  />
                )
              }
            })}
      </div>
    </Box>
  )
}
