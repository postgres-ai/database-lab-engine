import { makeStyles } from '@material-ui/core'
import ExplainVisualization from 'components/ExplainVisualization/ExplainVisualization'

export const ExplainVisualizationWrapper = () => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        width: '100%',
        [theme.breakpoints.down('sm')]: {
          maxWidth: 'calc(100vw - 40px)',
        },
        [theme.breakpoints.up('md')]: {
          maxWidth: 'calc(100vw - 220px)',
        },
        [theme.breakpoints.up('lg')]: {
          maxWidth: 'calc(100vw - 220px)',
        },
        minHeight: '100%',
        zIndex: 1,
        position: 'relative',
      },
      pointerLink: {
        cursor: 'pointer',
      },
      breadcrumbPaper: {
        marginBottom: 15,
      },
      planTextField: {
        '& .MuiInputBase-root': {
          width: '100%',
        },
        display: 'block',
        width: '100%',
      },
      appBar: {
        position: 'relative',
      },
      title: {
        marginLeft: theme.spacing(2),
        flex: 1,
        fontSize: '16px',
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
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <ExplainVisualization classes={classes} />
}
