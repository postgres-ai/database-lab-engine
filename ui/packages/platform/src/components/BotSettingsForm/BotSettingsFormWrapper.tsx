import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import OrgForm from 'components/OrgForm/OrgForm'
import BotSettingsForm from "./BotSettingsForm";

export interface BotSettingsFormProps {
  mode?: string | undefined
  project?: string | undefined
  org?: string | number
  orgId?: number
  orgPermissions?: {
    settingsOrganizationUpdate?: boolean
  }
}

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
    instructionsField: {
      ...styles.inputField,
      maxWidth: 450,
    },
    selectField: {
      maxWidth: 450,
      marginTop: 20,
      '& .MuiInputLabel-formControl': {
        transform: 'none',
        position: 'static'
      }
    },
    updateButtonContainer: {
      marginTop: 20,
      textAlign: 'left',
    },
    errorMessage: {
      color: 'red',
    },
  },
  { index: 1 },
)

export const BotSettingsFormWrapper = (props: BotSettingsFormProps) => {
  const classes = useStyles()

  return <BotSettingsForm {...props} classes={classes} />
}
