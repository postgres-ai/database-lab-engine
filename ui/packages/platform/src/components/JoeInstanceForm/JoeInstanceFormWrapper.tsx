import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { RouteComponentProps } from 'react-router'
import JoeInstanceForm from 'components/JoeInstanceForm/JoeInstanceForm'

export interface JoeInstanceFormProps {
  project: string | undefined
  orgId: number
  org: string | number
  history: RouteComponentProps['history']
  orgPermissions?: {
    joeInstanceCreate?: boolean
  }
}

export const JoeInstanceFormWrapper = (props: JoeInstanceFormProps) => {
  const useStyles = makeStyles(
    {
      textField: {
        ...styles.inputField,
        maxWidth: 400,
      },
      errorMessage: {
        color: 'red',
        marginTop: 10,
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

  return <JoeInstanceForm {...props} classes={classes} />
}
