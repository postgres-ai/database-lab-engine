import { makeStyles } from '@material-ui/core'
import { theme } from '@postgres.ai/shared/styles/theme'
import { styles } from '@postgres.ai/shared/styles/styles'
import CheckupAgentForm from 'components/CheckupAgentForm/CheckupAgentForm'

export interface CheckupAgentFormProps {
  orgId: number
}

export const CheckupAgentFormWrapper = (props: CheckupAgentFormProps) => {
  const useStyles = makeStyles(
    (muiTheme) => ({
      root: {
        'min-height': '100%',
        'z-index': 1,
        position: 'relative',
        [muiTheme.breakpoints.down('sm')]: {
          maxWidth: '100vw',
        },
        [muiTheme.breakpoints.up('md')]: {
          maxWidth: 'calc(100vw - 200px)',
        },
        [muiTheme.breakpoints.up('lg')]: {
          maxWidth: 'calc(100vw - 200px)',
        },
        '& h2': {
          ...theme.typography.h2,
        },
        '& h3': {
          ...theme.typography.h3,
        },
        '& h4': {
          ...theme.typography.h4,
        },
        '& .MuiExpansionPanelSummary-root.Mui-expanded': {
          minHeight: 24,
        },
      },
      heading: {
        ...theme.typography.h3,
      },
      fieldValue: {
        display: 'inline-block',
        width: '100%',
      },
      tokenInput: {
        ...styles.inputField,
        margin: 0,
        'margin-top': 10,
        'margin-bottom': 10,
      },
      textInput: {
        ...styles.inputField,
        margin: 0,
        marginTop: 0,
        marginBottom: 10,
      },
      hostItem: {
        marginRight: 10,
        marginBottom: 5,
        marginTop: 5,
      },
      fieldRow: {
        marginBottom: 10,
        display: 'block',
      },
      fieldBlock: {
        width: '100%',
        'max-width': 600,
        'margin-bottom': 15,
        '& > div.MuiFormControl- > label': {
          fontSize: 20,
        },
        '& input, & .MuiOutlinedInput-multiline': {
          padding: 13,
        },
      },
      relativeFieldBlock: {
        marginBottom: 10,
        marginRight: 20,
        position: 'relative',
      },
      addTokenButton: {
        marginLeft: 10,
        marginTop: 10,
      },
      code: {
        width: '100%',
        'margin-top': 0,
        '& > div': {
          paddingTop: 12,
        },
        'background-color': 'rgb(246, 248, 250)',
        '& > div > textarea': {
          fontFamily:
            '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
            ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
          color: 'black',
          fontSize: 14,
        },
      },
      codeBlock: {
        fontFamily:
          '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
          ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
        width: '100%',
        padding: 3,
        marginTop: 0,
        border: 'rgb(204, 204, 204);',
        borderRadius: 3,
        color: 'black',
        backgroundColor: 'rgb(246, 248, 250)',
      },
      details: {
        display: 'block',
      },
      copyButton: {
        position: 'absolute',
        top: 6,
        right: 6,
        fontSize: 20,
      },
      relativeDiv: {
        position: 'relative',
      },
      radioButton: {
        '& > label > span.MuiFormControlLabel-label': {
          fontSize: '0.9rem',
        },
      },
      legend: {
        fontSize: '10px',
      },
      advancedExpansionPanelSummary: {
        'justify-content': 'left',
        '& div.MuiExpansionPanelSummary-content': {
          'flex-grow': 0,
        },
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <CheckupAgentForm {...props} classes={classes} />
}
