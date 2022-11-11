import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import DbLabInstances from 'components/DbLabInstances/DbLabInstances'
import { RouteComponentProps } from 'react-router'
import { colors } from '@postgres.ai/shared/styles/colors'

export interface DbLabInstancesProps {
  orgId: number
  org: string | number
  project: string | undefined
  projectId: string | number | undefined
  history: RouteComponentProps['history']
  match: {
    params: {
      projectId?: string | number | undefined
    }
  }
  orgPermissions: {
    dblabInstanceCreate?: boolean
    dblabInstanceDelete?: boolean
    dblabInstanceList?: boolean
  }
}

export const DbLabInstancesWrapper = (props: DbLabInstancesProps) => {
  const useStyles = makeStyles(
    {
      root: {
        ...(styles.root as Object),
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
      inTableProgress: {
        width: '30px!important',
        height: '30px!important',
      },
      warningIcon: {
        color: colors.state.warning,
        fontSize: '1.2em',
        position: 'absolute',
        marginLeft: 5,
      },
      tooltip: {
        fontSize: '10px!important',
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <DbLabInstances {...props} classes={classes} />
}
