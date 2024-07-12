import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import DbLabInstances from 'components/DbLabInstances/DbLabInstances'
import { RouteComponentProps } from 'react-router'
import { colors } from '@postgres.ai/shared/styles/colors'
import { OrgPermissions } from 'components/types'

export interface DbLabInstancesProps {
  orgId: number
  org: string | number
  project: string | undefined
  projectId: string | number | undefined
  history: RouteComponentProps['history']
  match: {
    params: {
      project?: string
      projectId?: string | number | undefined
      org?: string
    }
  }
  orgPermissions: OrgPermissions
}

export const DbLabInstancesWrapper = (props: DbLabInstancesProps) => {
  const useStyles = makeStyles(
    {
      root: {
        ...(styles.root as Object),
        display: 'flex',
        flexDirection: 'column',
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
      timeLabel: {
        lineHeight: '16px',
        fontSize: 12,
        cursor: 'pointer',
      },
      buttonContainer: {
        display: 'flex',
        gap: 10,
      },
      flexContainer: {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: '100%',
        height: '100%',
        gap: 40,
        marginTop: '20px',

        '& > div': {
          maxWidth: '300px',
          width: '100%',
          height: '100%',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          border: '1px solid #e0e0e0',
          padding: '20px',
          borderRadius: '4px',
          cursor: 'pointer',
          fontSize: '15px',
          transition: 'border 0.3s ease-in-out',

          '&:hover': {
            border: '1px solid #FF6212',
          },
        },
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <DbLabInstances {...props} classes={classes} />
}
