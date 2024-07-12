import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { RouteComponentProps } from 'react-router'
import OrgSettings from 'components/OrgMembers/OrgMembers'
import { OrgPermissions } from 'components/types'

export interface OrgSettingsProps {
  project: string | undefined
  history: RouteComponentProps['history']
  org: string | number
  orgId: number
  orgPermissions: OrgPermissions
  env: {
    data: {
      info: {
        id: number | null
      }
    }
  }
}

export const OrgMembersWrapper = (props: OrgSettingsProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        ...(styles.root as Object),
        display: 'flex',
        flexDirection: 'column',
        paddingBottom: '20px',
      },
      container: {
        display: 'flex',
        flexWrap: 'wrap',
      },
      textField: {
        marginLeft: theme.spacing(1),
        marginRight: theme.spacing(1),
        width: '80%',
      },
      actionCell: {
        textAlign: 'right',
        padding: 0,
        paddingRight: 16,
      },
      iconButton: {
        margin: '-12px',
        marginLeft: 5,
      },
      inTableProgress: {
        width: '15px!important',
        height: '15px!important',
        marginLeft: 5,
        verticalAlign: 'middle',
      },
      roleSelector: {
        height: 24,
        width: 190,
        '& svg': {
          top: 5,
          right: 3,
        },
        '& .MuiSelect-select': {
          padding: 8,
          paddingRight: 20,
        },
      },
      roleSelectorItem: {
        fontSize: 14,
      },
      bottomSpace: {
        ...styles.bottomSpace,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <OrgSettings {...props} classes={classes} />
}
