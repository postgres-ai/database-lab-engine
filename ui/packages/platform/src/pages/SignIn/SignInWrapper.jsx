import { makeStyles } from '@material-ui/core'
import SignIn from 'pages/SignIn'

export const SignInWrapper = () => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        fontFamily: 'dinpro-regular,sans-serif',
        backgroundColor: '#fbfbfb',
        overflow: 'auto',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 40,
        [theme.breakpoints.down('xs')]: {
          padding: '40px 24px',
        },
      },

      form: {
        flex: '0 1 400px',
        background: '#fff',
        border: '1px solid #c9d8db',
        margin: 'auto',
        borderRadius: '3px',
        padding: 40,
        [theme.breakpoints.down('xs')]: {
          padding: 24,
        },
      },

      titleLink: {
        textDecoration: 'none',
        cursor: 'pointer',
      },

      title: {
        fontFamily: 'dinpro-light,sans-serif',
        color: '#1a1a1a',
        fontSize: '3rem',
        fontWeight: 300,
        marginBottom: 32,
        marginTop: 0,
        textAlign: 'center',
        [theme.breakpoints.down('xs')]: {
          fontSize: '2rem',
          marginBottom: 24,
        },
      },

      button: {
        fontFamily: 'roboto,sans-serif',
        display: 'flex',
        height: 40,
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: 14,
        borderRadius: 5,
        backgroundColor: '#fff',
        color: 'rgba(0,0,0,.54)',
        border: '1px solid #cecece',
        fontWeight: 500,
        marginBottom: 16,
        textDecoration: 'none',
        padding: 12,
      },

      buttonText: {
        marginLeft: 24,
        whiteSpace: 'nowrap',
        flex: '0 0 131px',

        [theme.breakpoints.down('xs')]: {
          marginLeft: 16,
        },
      },

      terms: {
        fontFamily: 'inherit',
        paddingTop: 10,
        fontSize: 12,
        textAlign: 'center',
        lineHeight: '16px',
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <SignIn classes={classes} />
}
