import { makeStyles } from '@material-ui/core'
import JoeSessionCommand from 'pages/JoeSessionCommand'
import { styles } from '@postgres.ai/shared/styles/styles'

export const JoeSessionCommandWrapper = (props) => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        display: 'flex',
        'flex-direction': 'column',
        flex: '1 1 100%',

        '& h4': {
          marginTop: '2em',
          marginBottom: '0.7em',
        },
      },
      appBar: {
        position: 'relative',
      },
      title: {
        marginLeft: theme.spacing(2),
        flex: 1,
        fontSize: '16px',
      },
      actions: {
        display: 'flex',
      },
      visFrame: {
        height: '100%',
      },
      nextButton: {
        marginLeft: '10px',
      },
      flameGraphContainer: {
        padding: '20px',
      },
      bottomSpace: {
        ...styles.bottomSpace,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <JoeSessionCommand {...props} classes={classes} />
}
