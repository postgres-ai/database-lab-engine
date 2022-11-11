import { makeStyles } from '@material-ui/core'
import DisplayToken from 'components/DisplayToken/DisplayToken'
import { styles } from '@postgres.ai/shared/styles/styles'

export const DisplayTokenWrapper = () => {
  const useStyles = makeStyles(
    {
      textField: {
        ...styles.inputField,
        marginTop: 0,
      },
      input: {
        '&.MuiOutlinedInput-adornedEnd': {
          padding: 0,
        },
      },
      inputElement: {
        marginRight: '-8px',
      },
      inputAdornment: {
        margin: 0,
      },
      inputButton: {
        padding: '9px 10px',
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <DisplayToken classes={classes} />
}
