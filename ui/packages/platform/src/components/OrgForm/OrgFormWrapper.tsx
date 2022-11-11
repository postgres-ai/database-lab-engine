import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import OrgForm from 'components/OrgForm/OrgForm'

interface OrgFormProps {
  mode?: string | undefined
  project: string | undefined
  org: string | number
  orgId: number
  orgPermissions: {
    settingsOrganizationUpdate?: boolean
  }
}

export const OrgFormWrapper = (props: OrgFormProps) => {
  const useStyles = makeStyles(
    {
      container: {
        ...(styles.root as Object),
        display: 'flex',
        'flex-wrap': 'wrap',
        'min-height': 0,
        '&:not(:first-child)': {
          'margin-top': '20px',
        },
      },
      textField: {
        ...styles.inputField,
        maxWidth: 450,
      },
      onboardingField: {
        ...styles.inputField,
        maxWidth: 450,
        '& #onboardingText-helper-text': {
          marginTop: 5,
          fontSize: '10px',
          letterSpacing: 0,
        },
      },
      updateButtonContainer: {
        marginTop: 10,
        textAlign: 'left',
      },
      errorMessage: {
        color: 'red',
      },
      autoJoinCheckbox: {
        ...styles.checkbox,
        'margin-top': 5,
        'margin-bottom': 10,
        'margin-Left': 0,
        width: 300,
      },
      comment: {
        fontSize: 14,
        marginLeft: 0,
        width: '80%',
      },
      domainsHeader: {
        lineHeight: '42px',
      },
      editDomainField: {
        ...styles.inputField,
        'margin-left': 0,
        'margin-top': 10,
        width: 'calc(80% - 80px)',
        '& > div.MuiFormControl- > label': {
          fontSize: 20,
        },
        '& input, & .MuiOutlinedInput-multiline, & .MuiSelect-select': {
          padding: 13,
        },
      },
      domainButton: {
        marginTop: 11,
        marginLeft: 10,
        width: 70,
      },
      newDomainField: {
        ...styles.inputField,
        'margin-left': 0,
        'margin-top': 10,
        width: '80%',
        '& > div.MuiFormControl- > label': {
          fontSize: 20,
        },
        '& input, & .MuiOutlinedInput-multiline, & .MuiSelect-select': {
          padding: 13,
        },
      },
      tableCell: {
        padding: 7,
        lineHeight: '16px',
        fontSize: 14,
      },
      tableCellRight: {
        fontSize: 14,
        padding: 7,
        lineHeight: '16px',
        textAlign: 'right',
      },
      table: {
        width: '80%',
      },
      tableActionButton: {
        padding: 0,
        fontSize: 16,
      },
      supportLink: {
        cursor: 'pointer',
        fontWeight: 'bold',
        textDecoration: 'underline',
      },
      bottomSpace: {
        ...styles.bottomSpace,
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <OrgForm {...props} classes={classes} />
}
