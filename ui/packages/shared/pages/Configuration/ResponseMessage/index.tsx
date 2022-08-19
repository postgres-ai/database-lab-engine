import { makeStyles } from '@material-ui/core/styles'
import BlockIcon from '@material-ui/icons/Block'
import CheckCircleOutlineIcon from '@material-ui/icons/CheckCircleOutline'
import WarningIcon from '@material-ui/icons/Warning'

const useStyles = makeStyles({
  successIcon: {
    marginRight: 8,
    color: 'green',
  },
  success: {
    color: 'green',
    alignItems: 'center',
    display: 'flex',
  },
  errorIcon: {
    marginRight: 8,
    color: 'red',
  },
  error: {
    color: 'red',
    alignItems: 'center',
    display: 'flex',
  },
  warning: {
    color: '#FD8411',
    alignItems: 'center',
    display: 'flex',
  },
  warningIcon: {
    marginRight: 8,
    color: '#FD8411',
  },
})

export const ResponseMessage = ({
  type,
  message,
}: {
  type: string
  message: string | React.ReactNode | null
}) => {
  const classes = useStyles()

  return (
    <span
      className={
        type === 'success' || type === 'ok'
          ? classes.success
          : type === 'warning' || type === 'notice'
          ? classes.warning
          : classes.error
      }
    >
      {type === 'success' || type === 'ok' ? (
        <CheckCircleOutlineIcon className={classes.successIcon} />
      ) : type === 'warning' || type === 'notice' ? (
        <WarningIcon className={classes.warningIcon} />
      ) : (
        <BlockIcon className={classes.errorIcon} />
      )}
      {message}
    </span>
  )
}
