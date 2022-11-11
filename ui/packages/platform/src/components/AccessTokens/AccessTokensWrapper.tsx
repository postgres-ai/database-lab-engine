import { makeStyles } from '@material-ui/core'
import AccessTokens from 'components/AccessTokens/AccessTokens'
import { OrgPermissions } from 'components/types'
import { styles } from '@postgres.ai/shared/styles/styles'

export interface AccessTokensProps {
  project: string | undefined
  orgId: number
  org: string | number
  orgPermissions: OrgPermissions
}

export const AccessTokensWrapper = (props: AccessTokensProps) => {
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
        ...styles.inputField,
        maxWidth: 400,
        marginBottom: 15,
        marginRight: theme.spacing(1),
        marginTop: '16px',
      },
      nameField: {
        ...styles.inputField,
        maxWidth: 400,
        marginBottom: 15,
        width: '400px',
        marginRight: theme.spacing(1),
      },
      addTokenButton: {
        marginTop: 15,
        height: '33px',
        marginBottom: 10,
      },
      revokeButton: {
        paddingRight: 5,
        paddingLeft: 5,
        paddingTop: 3,
        paddingBottom: 3,
      },
      errorMessage: {
        color: 'red',
        width: '100%',
      },
      remark: {
        width: '100%',
        maxWidth: 960,
      },
      bottomSpace: {
        ...styles.bottomSpace,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <AccessTokens {...props} classes={classes} />
}
