import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { RouteComponentProps } from 'react-router'
import Report from 'components/Report/Report'

interface MatchParams {
  reportId: string
  type?: string
}

export interface ReportProps extends RouteComponentProps<MatchParams> {
  orgId: number
}
export const ReportWrapper = (props: ReportProps) => {
  const useStyles = makeStyles(
    {
      root: {
        ...(styles.root as Object),
        flex: '1 1 100%',
        display: 'flex',
        flexDirection: 'column',
      },
      cell: {
        '& > a': {
          color: 'black',
          textDecoration: 'none',
        },
        '& > a:hover': {
          color: 'black',
          textDecoration: 'none',
        },
      },
      horizontalMenuItem: {
        display: 'inline-block',
        marginRight: 10,
        'font-weight': 'normal',
        color: 'black',
        '&:visited': {
          color: 'black',
        },
      },
      activeHorizontalMenuItem: {
        display: 'inline-block',
        marginRight: 10,
        'font-weight': 'bold',
        color: 'black',
      },
      fileTypeMenu: {
        marginBottom: 10,
      },
      bottomSpace: {
        ...styles.bottomSpace,
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <Report {...props} classes={classes} />
}
