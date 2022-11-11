import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { RouteComponentProps } from 'react-router'
import DbLabSessions from 'components/DbLabSessions/DbLabSessions'

interface DbLabSessionsProps {
  org: string | number
  orgId: number
  history: RouteComponentProps['history']
}

export const DbLabSessionsWrapper = (props: DbLabSessionsProps) => {
  const useStyles = makeStyles(
    {
      root: {
        ...(styles.root as Object),
        paddingBottom: '20px',
        display: 'flex',
        flexDirection: 'column',
      },
      tableHead: {
        ...(styles.tableHead as Object),
        textAlign: 'left',
      },
      tableCell: {
        textAlign: 'left',
      },
      showMoreContainer: {
        marginTop: 20,
        textAlign: 'center',
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <DbLabSessions {...props} classes={classes} />
}
