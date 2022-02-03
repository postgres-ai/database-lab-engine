import { makeStyles } from '@material-ui/core'

import { WarningIcon } from '@postgres.ai/shared/icons/Warning'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { colors } from '@postgres.ai/shared/styles/colors'

export type Props = {
  value: 'refreshing' | 'active' | 'empty'
  hasWarning: boolean
}

const VALUE_TO_NAME: Record<Props['value'], string> = {
  refreshing: 'Data refreshing',
  active: 'Active',
  empty: 'Empty',
}

const VALUE_TO_DESC: Record<Props['value'], string> = {
  refreshing: 'Data retrieval is in progress',
  active: 'Disk is ready to provision clones',
  empty: 'Disk is emptied and ready for a data retrieval',
}

const useStyles = makeStyles({
  root: {
    display: 'flex',
    color: colors.white,
  },
  warning: {
    display: 'flex',
    flex: '0 0 16px',
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: '4px',
    height: '16px',
    background: colors.state.warning,
    borderRadius: '3px',
  },
  icon: {
    height: '10px',
  },
  label: {
    fontSize: '12px',
    padding: '1px 6px',
    borderRadius: '3px',
    flex: '0 0 auto',
    background: (props: Props) => {
      if (props.value === 'refreshing') return colors.state.notice
      if (props.value === 'active') return colors.state.ok
      return colors.state.unknown
    },
  },
})

export const Status = (props: Props) => {
  const { value, hasWarning } = props

  const classes = useStyles(props)

  return (
    <div className={classes.root}>
      {hasWarning && (
        <div className={classes.warning}>
          <WarningIcon className={classes.icon} />
        </div>
      )}
      <Tooltip content={VALUE_TO_DESC[value]}>
        <div className={classes.label}>{VALUE_TO_NAME[value]}</div>
      </Tooltip>
    </div>
  )
}
