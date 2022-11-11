import { makeStyles } from '@material-ui/core'
import { colors } from '@postgres.ai/shared/styles/colors'
import { styles } from '@postgres.ai/shared/styles/styles'
import JoeInstance from 'pages/JoeInstance'

export const JoeInstanceWrapper = (props) => {
  const useStyles = makeStyles(
    (theme) => ({
      messageArtifacts: {
        'box-shadow': 'none',
        'min-height': 20,
        'font-size': '14px',
        'margin-top': '15px!important',
        '& > .MuiExpansionPanelSummary-root': {
          minHeight: 20,
          backgroundColor: '#FFFFFF',
          border: '1px solid',
          borderColor: colors.secondary2.main,
          width: 'auto',
          color: colors.secondary2.main,
          borderRadius: 3,
          overflow: 'hidden',
        },
        '& > .MuiExpansionPanelSummary-root.Mui-expanded': {
          minHeight: 20,
        },
        '& .MuiCollapse-hidden': {
          height: '0px!important',
        },
      },
      advancedExpansionPanelSummary: {
        'justify-content': 'left',
        padding: '0px',
        'background-color': '#FFFFFF',
        width: 'auto',
        color: colors.secondary2.main,
        'border-radius': '3px',
        'padding-left': '15px',
        display: 'inline-block',

        '& div.MuiExpansionPanelSummary-content': {
          'flex-grow': 'none',
          margin: 0,
          '& > p.MuiTypography-root.MuiTypography-body1': {
            'font-size': '12px!important',
          },
          display: 'inline-block',
        },
        '& > .MuiExpansionPanelSummary-expandIcon': {
          marginRight: 0,
          padding: '5px!important',
          color: colors.secondary2.main,
        },
      },
      advancedExpansionPanelDetails: {
        padding: 5,
        'padding-left': 0,
        'padding-right': 0,
        'font-size': '14px!important',
        '& > p': {
          'font-size': '14px!important',
        },
      },
      messageArtifactLink: {
        cursor: 'pointer',
        color: '#0000ee',
      },
      messageArtifactTitle: {
        fontWeight: 'normal',
        marginBottom: 0,
      },
      messageArtifact: {
        'box-shadow': 'none',
        'min-height': 20,
        'font-size': '14px',
        '&.MuiExpansionPanel-root.Mui-expanded': {
          marginBottom: 0,
          marginTop: 0,
        },
        '& > .MuiExpansionPanelSummary-root': {
          minHeight: 20,
          backgroundColor: '#FFFFFF',
          border: 'none',
          width: 'auto',
          color: colors.secondary2.darkDark,
          borderRadius: 3,
          overflow: 'hidden',
          marginBottom: 0,
        },
        '& > .MuiExpansionPanelSummary-root.Mui-expanded': {
          minHeight: 20,
        },
      },
      messageArtifactExpansionPanelSummary: {
        'justify-content': 'left',
        padding: '5px',
        'background-color': '#FFFFFF',
        width: 'auto',
        color: colors.secondary2.darkDark,
        'border-radius': '3px',
        'padding-left': '0px',
        display: 'inline-block',

        '& div.MuiExpansionPanelSummary-content': {
          'flex-grow': 'none',
          margin: 0,
          '& > p.MuiTypography-root.MuiTypography-body1': {
            'font-size': '12px!important',
          },
          display: 'inline-block',
        },
        '& > .MuiExpansionPanelSummary-expandIcon': {
          marginRight: 0,
          padding: '5px!important',
          color: colors.secondary2.darkDark,
        },
      },
      messageArtifactExpansionPanelDetails: {
        padding: 5,
        'padding-right': 0,
        'padding-left': 0,
        'font-size': '14px!important',
        '& > p': {
          'font-size': '14px!important',
        },
      },
      code: {
        width: '100%',
        'margin-top': 0,
        '& > div': {
          paddingTop: 8,
          padding: 8,
        },
        'background-color': 'rgb(246, 248, 250)',
        '& > div > textarea': {
          fontFamily:
            '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
            ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
          color: 'black',
          fontSize: '13px',
        },
      },
      messageArtifactsContainer: {
        width: '100%',
      },
      heading: {
        fontWeight: 'normal',
        fontSize: 12,
        color: colors.secondary2.main,
      },
      message: {
        padding: 10,
        'padding-left': 60,
        position: 'relative',

        '& .markdown pre': {
          [theme.breakpoints.down('sm')]: {
            display: 'inline-block',
            minWidth: '100%',
            width: 'auto',
          },
          [theme.breakpoints.up('md')]: {
            display: 'block',
            maxWidth: 'auto',
            width: 'auto',
          },
          [theme.breakpoints.up('lg')]: {
            display: 'block',
            maxWidth: 'auto',
            width: 'auto',
          },
        },
        '&:hover $repeatCmdButton': {
          display: 'inline-block',
        },
      },
      messageAvatar: {
        top: '10px',
        left: '15px',
        position: 'absolute',
      },
      messageAuthor: {
        fontSize: 14,
        fontWeight: 'bold',
      },
      messageTime: {
        display: 'inline-block',
        marginLeft: 10,
        fontSize: '14px',
        color: colors.pgaiDarkGray,
      },
      sqlCode: {
        padding: 0,
        border: 'none!important',
        'margin-top': 0,
        'margin-bottom': 0,
        '& > .MuiInputBase-fullWidth > fieldset': {
          border: 'none!important',
        },
        '& > .MuiInputBase-fullWidth': {
          padding: 5,
        },
        '& > div > textarea': {
          fontFamily:
            '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
            ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
          color: 'black',
          fontSize: '14px',
        },
      },
      messageStatusContainer: {
        marginTop: 15,
      },
      messageStatus: {
        border: '1px solid #CCCCCC',
        borderColor: colors.pgaiLightGray,
        borderRadius: 3,
        display: 'inline-block',
        padding: 3,
        fontSize: '12px',
        lineHeight: '12px',
        paddingLeft: 6,
        paddingRight: 6,
      },
      messageStatusIcon: {
        '& svg': {
          marginBottom: -2,
        },
        'margin-right': 3,
      },
      messageProgress: {
        '& svg': {
          marginBottom: -2,
          display: 'inline-block',
        },
      },
      actions: {
        display: 'flex',
        justifyContent: 'space-between',
      },
      channelsList: {
        ...styles.inputField,
        marginBottom: 0,
        marginRight: '8px',
        flex: '0 1 240px',
      },
      clearChatButton: {
        flex: '0 0 auto',
      },
      repeatCmdButton: {
        fontSize: '12px',
        padding: '2px 5px',
        marginLeft: 10,
        display: 'none',
        lineHeight: '14px',
        marginTop: '-4px',
      },
      messageHeader: {
        height: '18px',
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <JoeInstance {...props} classes={classes} />
}
