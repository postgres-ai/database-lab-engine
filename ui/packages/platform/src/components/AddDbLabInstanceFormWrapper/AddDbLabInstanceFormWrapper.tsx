import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import AddDbLabInstanceForm from 'components/AddDbLabInstanceFormWrapper/AddDblabInstanceForm'
import { OrgPermissions } from 'components/types'
import { RouteComponentProps } from 'react-router'

export interface DbLabInstanceFormProps {
  edit?: boolean
  orgId: number
  project: string | undefined
  history: RouteComponentProps['history']
  orgPermissions: OrgPermissions
}

export const AddDbLabInstanceFormWrapper = (props: DbLabInstanceFormProps) => {
  const useStyles = makeStyles(
    {
      textField: {
        ...styles.inputField,
        maxWidth: 400,
      },
      errorMessage: {
        marginTop: 10,
        color: 'red',
      },
      fieldBlock: {
        width: '100%',
      },
      urlOkIcon: {
        color: 'green',
      },
      urlOk: { display: 'flex', gap: 5, alignItems: 'center', color: 'green' },
      urlTextMargin: {
        marginTop: 10,
      },
      urlFailIcon: {
        color: 'red',
      },
      urlFail: {
        display: 'flex',
        gap: 5,
        alignItems: 'center',
        color: 'red',
      },
      warning: {
        color: '#801200',
        fontSize: '0.9em',
      },
      warningIcon: {
        color: '#801200',
        fontSize: '1.2em',
        position: 'relative',
        marginBottom: -3,
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <AddDbLabInstanceForm {...props} classes={classes} />
}
