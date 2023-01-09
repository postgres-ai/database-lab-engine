import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { RouteComponentProps } from 'react-router'
import { OrgPermissions } from 'components/types'
import InviteForm from 'components/AddMemberForm/AddMemberForm'

export interface InviteFormProps {
  org: string | number
  orgId: number
  history: RouteComponentProps['history']
  project: string | undefined
  orgPermissions: OrgPermissions
}

export const AddMemberFormWrapper = (props: InviteFormProps) => {
  const useStyles = makeStyles(
    {
      container: {
        display: 'flex',
        flexWrap: 'wrap',
      },
      textField: {
        ...styles.inputField,
        maxWidth: 400,
      },
      dense: {
        marginTop: 10,
      },
      errorMessage: {
        color: 'red',
      },
      button: {
        marginTop: 17,
        display: 'inline-block',
        marginLeft: 7,
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <InviteForm {...props} classes={classes} />
}
