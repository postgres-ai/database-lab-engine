import { makeStyles } from '@material-ui/core'
import Error from 'components/Error/Error'

export interface ErrorProps {
  code?: number
  message?: string
}

export const ErrorWrapper = (props: ErrorProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      paper: theme.mixins.gutters({
        paddingTop: 16,
        paddingBottom: 16,
        marginTop: 0,
      }),
      errorHead: {
        color: '#c00111',
        fontWeight: 'bold',
        fontSize: '16px',
      },
      errorText: {
        color: '#c00111',
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <Error {...props} classes={classes} />
}
