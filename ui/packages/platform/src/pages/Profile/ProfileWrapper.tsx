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
      formControlLabel: {
        marginLeft: theme.spacing(0),
        marginRight: theme.spacing(1),
      },
      formControlLabelCheckbox: {
        '& svg': {
          fontSize: 18
        }
      },
      updateButtonContainer: {
        marginTop: theme.spacing(3),
        marginLeft: theme.spacing(1),
        marginRight: theme.spacing(1),
      },
      label: {
        marginTop: theme.spacing(2),
        marginBottom: theme.spacing(1),
        marginLeft: theme.spacing(1),
        marginRight: theme.spacing(1),
        color: '#000!important',
        fontWeight: 'bold',
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
