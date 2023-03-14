import classNames from 'classnames'
import Box from '@mui/material/Box'
import { TextField, Chip, makeStyles } from '@material-ui/core'

import { Select } from '@postgres.ai/shared/components/Select'
import { InfoIcon } from '@postgres.ai/shared/icons/Info'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'

import { uniqueChipValue } from '../utils'

import styles from '../styles.module.scss'

const useStyles = makeStyles(
  {
    textField: {
      '& .MuiOutlinedInput-notchedOutline': {
        borderColor: '#000',
      },
    },
    selectField: {
      marginTop: '0',
      '& .MuiInputBase-root': {
        padding: '6px',
      },

      '& .MuiSelect-select:focus': {
        backgroundColor: 'inherit',
      },
    },
    label: {
      display: 'block',
    },
    error: {
      color: '#f44336',
    },
  },
  { index: 1 },
)

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
  disabled: boolean | undefined
}) => {
  const classes = useStyles()

  return (
    <Box
      mt={2}
      mb={2}
      display="flex"
      flexDirection="column"
      justifyContent="flex-start"
      alignItems="flex-start"
      gap="5px"
    >
      <label className={classNames(error && classes.error, classes.label)}>
        {label}
      </label>
      <Box display="flex" alignItems="center" width="100%">
        <TextField
          className={classNames(
            !disabled && classes.textField,
            styles.textField,
          )}
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
  handleDeleteChip,
}: {
  value: string
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  tooltipText: () => React.ReactNode
  handleDeleteChip: (
    event: React.FormEvent<HTMLInputElement>,
    uniqueValue: string,
    label: string,
  ) => void
  label: string
  id: string
  disabled: boolean | undefined
}) => {
  const classes = useStyles()

  return (
    <Box
      mt={2}
      mb={1}
      display="flex"
      flexDirection="column"
      justifyContent="flex-start"
      alignItems="flex-start"
      gap="5px"
    >
      <label className={classes.label}>{label}</label>
      <Box display="flex" alignItems="center" width="100%">
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
      {value && (
        <div className={styles.chipContainer}>
          {uniqueChipValue(value)
            .split(' ')
            .map((uniqueValue, index) => {
              if (uniqueValue !== '') {
                return (
                  <Chip
                    key={index}
                    className={styles.chip}
                    label={uniqueValue}
                    disabled={disabled}
                    onDelete={(event) =>
                      handleDeleteChip(event, uniqueValue, id)
                    }
                    color="primary"
                  />
                )
              }
            })}
        </div>
      )}
    </Box>
  )
}

export const SelectWithTooltip = ({
  value,
  label,
  error,
  onChange,
  tooltipText,
  disabled,
  items,
}: {
  value: string
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  tooltipText: () => React.ReactNode
  label: string
  error?: boolean
  disabled: boolean | undefined
  items: { value: string; children: React.ReactNode }[]
}) => {
  const classes = useStyles()

  return (
    <Box
      mt={1}
      display="flex"
      flexDirection="column"
      justifyContent="flex-start"
      alignItems="flex-start"
      gap="5px"
    >
      <label className={classNames(error && classes.error, classes.label)}>
        {label}
      </label>
      <Box display="flex" alignItems="center" width="100%">
        <Select
          className={classNames(
            classes.selectField,
            !disabled && classes.textField,
            styles.textField,
          )}
          label=""
          error={error}
          value={value}
          disabled={disabled}
          onChange={onChange}
          items={items}
        />
        <Tooltip interactive content={<p>{tooltipText()}</p>}>
          <InfoIcon className={styles.infoIcon} />
        </Tooltip>
      </Box>
    </Box>
  )
}
