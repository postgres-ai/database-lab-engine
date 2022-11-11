import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { RouteComponentProps } from 'react-router'
import Reports from 'components/Reports/Reports'

export interface ReportsProps extends RouteComponentProps {
  projectId: string | undefined | number
  project: string | undefined
  orgId: number
  org: string | number
  orgPermissions: {
    checkupReportConfigure?: boolean
    checkupReportDelete?: boolean
  }
}

export const ReportsWrapper = (props: ReportsProps) => {
  const useStyles = makeStyles(
    {
      root: {
        ...(styles.root as Object),
        paddingBottom: '20px',
        display: 'flex',
        flexDirection: 'column',
      },
      stubContainer: {
        marginTop: '10px',
      },
      filterSelect: {
        ...styles.inputField,
        width: 150,
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
      tableHead: {
        ...(styles.tableHead as Object),
      },
      tableHeadActions: {
        ...(styles.tableHeadActions as Object),
      },
      checkboxTableCell: {
        width: '30px',
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <Reports {...props} classes={classes} />
}
