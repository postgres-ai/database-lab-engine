import { makeStyles } from '@material-ui/core'
import Notification from 'components/Notification/Notification'

export const NotificationWrapper = () => {
  const useStyles = makeStyles(
    {
      defaultNotification: {},
      errorNotification: {
        '& > div.MuiSnackbarContent-root': {
          backgroundColor: '#f44336!important',
        },
      },
      informationNotification: {
        '& > div.MuiSnackbarContent-root': {
          backgroundColor: '#2196f3!important',
        },
      },
      warningNotification: {
        '& > div.MuiSnackbarContent-root': {
          backgroundColor: '#ff9800!important',
        },
      },
      successNotification: {
        '& > div.MuiSnackbarContent-root': {
          backgroundColor: '#4caf50!important',
        },
      },
      svgIcon: {
        marginBottom: -2,
        marginRight: 5,
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <Notification classes={classes} />
}
