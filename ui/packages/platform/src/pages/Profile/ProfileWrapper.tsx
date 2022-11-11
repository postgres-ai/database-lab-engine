import { makeStyles } from '@material-ui/core'
import Profile from 'pages/Profile'

export const ProfileWrapper = () => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        // To be consistent with parent layout.
        // TODO(anton): rewrite parent layout.
        paddingBottom: 'inherit',
      },
      container: {
        display: 'flex',
        flexWrap: 'wrap',
      },
      textField: {
        marginLeft: theme.spacing(1),
        marginRight: theme.spacing(1),
      },
      dense: {
        marginTop: 16,
      },
      menu: {
        width: 2100,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <Profile classes={classes} />
}
