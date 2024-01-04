import classNames from 'classnames'
import Box from '@mui/material/Box'
import { TextField, Chip, makeStyles } from '@material-ui/core'

import { Select } from '@postgres.ai/shared/components/Select'
import { InfoIcon } from '@postgres.ai/shared/icons/Info'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { Spinner } from '@postgres.ai/shared/components/Spinner'

import { uniqueChipValue } from '../utils'

import styles from '../styles.module.scss'

const useStyles = makeStyles(
  {
    textField: {
      '& .MuiOutlinedInput-notchedOutline': {
        borderColor: '#000 !important',
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
    infoIcon: {
      position: 'relative',
    },
    absoluteSpinner: {
      position: 'absolute',
      left: 'calc(50% - 40px)',
      top: 'calc(50% - 5px)',
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
  type,
}: {
  value?: string
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  tooltipText: () => React.ReactNode
  label: string
  error?: string
  disabled: boolean | undefined
  type?: string
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
          type={type || 'text'}
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
          multiline={type === 'textarea'}
          spellCheck={false}
        />
        <Tooltip interactive content={<p>{tooltipText()}</p>}>
          <div className={styles.infoIcon}>
            <InfoIcon />
          </div>
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
          <div className={styles.infoIcon}>
            <InfoIcon />
          </div>
        </Tooltip>
      </Box>
      <div className={styles.chipContainer}>
        {value &&
          uniqueChipValue(value)
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
  loading,
  items,
}: {
  value: string
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  tooltipText: () => React.ReactNode
  label: string
  error?: boolean
  disabled: boolean | undefined
  loading?: boolean
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
      position="relative"
      gap="5px"
    >
      <label className={classNames(error && classes.error, classes.label)}>
        {label}
      </label>
      <Box display="flex" alignItems="center" width="100%">
        {loading && <Spinner className={classes.absoluteSpinner} />}
        <Select
          className={classNames(
            classes.selectField,
            !disabled && classes.textField,
            styles.textField,
          )}
          label={''}
          error={error}
          value={value}
          disabled={disabled}
          onChange={onChange}
          items={items}
        />
        <Tooltip interactive content={<p>{tooltipText()}</p>}>
          <div className={styles.infoIcon}>
            <InfoIcon />
          </div>
        </Tooltip>
      </Box>
    </Box>
  )
}
