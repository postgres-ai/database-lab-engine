import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import DbLabInstanceForm from 'components/DbLabInstanceForm/DbLabInstanceForm'
import { RouteComponentProps } from 'react-router'

export interface DbLabInstanceFormProps {
  edit?: boolean
  orgId: number
  project: string | undefined
  history: RouteComponentProps['history']
  orgPermissions: {
    dblabInstanceCreate?: boolean
  }
}

export const DbLabInstanceFormWrapper = (props: DbLabInstanceFormProps) => {
  const useStyles = makeStyles(
    {
      textField: {
        ...styles.inputField,
        maxWidth: 400,
      },
      errorMessage: {
        color: 'red',
      },
      fieldBlock: {
        width: '100%',
      },
      urlOkIcon: {
        marginBottom: -5,
        marginLeft: 10,
        color: 'green',
      },
      urlOk: {
        color: 'green',
      },
      urlFailIcon: {
        marginBottom: -5,
        marginLeft: 10,
        color: 'red',
      },
      urlFail: {
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

  return <DbLabInstanceForm {...props} classes={classes} />
}
