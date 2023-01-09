import { makeStyles } from '@material-ui/core'
import LoginDialog from 'components/LoginDialog/LoginDialog'

export const LoginDialogWrapper = () => {
  const useStyles = makeStyles(
    {
      error: {
        color: 'red',
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <LoginDialog classes={classes} />
}
