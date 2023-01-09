import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { colors } from '@postgres.ai/shared/styles/colors'
import { RouteComponentProps } from 'react-router'
import JoeInstances from 'components/JoeInstances/JoeInstances'

export interface JoeInstancesProps {
  orgId: number
  org: string | number
  project: string | undefined
  projectId: number | string | undefined
  history: RouteComponentProps['history']
  orgPermissions?: {
    joeInstanceCreate?: boolean
    joeInstanceDelete?: boolean
  }
  match: {
    params: {
      projectId?: number
    }
  }
}
export const JoeInstancesWrapper = (props: JoeInstancesProps) => {
  const useStyles = makeStyles(
    {
      root: {
        ...(styles.root as Object),
        paddingBottom: '20px',
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
      warningIcon: {
        color: colors.state.warning,
        fontSize: '1.2em',
        position: 'absolute',
        marginLeft: 5,
      },
      toolTip: {
        fontSize: '10px!important',
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <JoeInstances {...props} classes={classes} />
}
