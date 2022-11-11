import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { colors } from '@postgres.ai/shared/styles/colors'
import ShareUrlDialog from 'components/ShareUrlDialog/ShareUrlDialog'

export const ShareUrlDialogWrapper = () => {
  const useStyles = makeStyles(
    () => ({
      textField: {
        ...styles.inputField,
        marginTop: '0px',
        width: 480,
      },
      copyButton: {
        marginTop: '-3px',
        fontSize: '20px',
      },
      dialog: {},
      remark: {
        fontSize: 12,
        lineHeight: '12px',
        color: colors.state.warning,
        paddingLeft: 20,
      },
      remarkIcon: {
        display: 'block',
        height: '20px',
        width: '22px',
        float: 'left',
        paddingTop: '5px',
      },
      urlContainer: {
        marginTop: 10,
        paddingLeft: 22,
      },
      radioLabel: {
        fontSize: 12,
      },
      dialogContent: {
        paddingTop: 10,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <ShareUrlDialog classes={classes} />
}
