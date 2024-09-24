import { makeStyles } from '@material-ui/core'
import { colors } from '@postgres.ai/shared/styles/colors'
import { styles } from '@postgres.ai/shared/styles/styles'
import DbLabSession from 'components/DbLabSession/DbLabSession'
import { OrgPermissions } from 'components/types'
import { RouteComponentProps } from 'react-router'

interface MatchParams {
  sessionId: string
}

export interface DbLabSessionProps extends RouteComponentProps<MatchParams> {
  orgPermissions: OrgPermissions
}

export interface ErrorProps {
  code?: number
  message?: string
}

export const DbLabSessionWrapper = (props: DbLabSessionProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        ...(styles.root as Object),
        flex: '1 1 100%',
        display: 'flex',
        flexDirection: 'column',
      },
      summary: {
        marginTop: 20,
        marginBottom: 20,
      },
      paramTitle: {
        display: 'inline-block',
        width: 200,
      },
      sectionHeader: {
        fontWeight: 600,
        display: 'block',
        paddingBottom: 10,
        marginBottom: 10,
        borderBottom: '1px solid ' + colors.consoleStroke,
      },
      logContainer: {
        backgroundColor: 'black',
        color: 'white',
        fontFamily:
          '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
          ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
        fontSize: '13px',
        maxHeight: 'calc(100vh - 120px)',
        overflowY: 'auto',
        width: '100%',
        '& > div': {
          overflowWrap: 'anywhere',
        },
      },
      artifactContainer: {
        backgroundColor: 'black',
        color: 'white',
        fontFamily:
          '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
          ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
        fontSize: '13px',
        maxHeight: 'calc(100vh - 300px)',
        width: '100%',
        whiteSpace: 'break-spaces',
        overflowWrap: 'anywhere',
        overflow: 'auto',
      },
      showMoreContainer: {
        marginTop: 20,
        textAlign: 'center',
      },
      link: {
        color: colors.secondary2.main,
        '&:visited': {
          color: colors.secondary2.main,
        },
        '&:hover': {
          color: colors.secondary2.main,
        },
        '&:active': {
          color: colors.secondary2.main,
        },
      },
      checkStatusColumn: {
        display: 'block',
        width: 80,
        marginTop: 10,
        height: 30,
        float: 'left',
      },
      checkDescriptionColumn: {
        display: 'inline-block',
      },
      checkDetails: {
        clear: 'both',
        display: 'block',
        color: colors.pgaiDarkGray,
      },
      checkListItem: {
        marginBottom: 10,
        minHeight: 30,
      },
      cfgListItem: {
        marginBottom: 5,
      },
      expansionPanel: {
        marginTop: '5px!important',
        borderRadius: '0px!important',
      },
      expansionPanelSummary: {
        display: 'inline-block',
        padding: '5px',
        paddingLeft: '12px',
        minHeight: '30px',
        lineHeight: '30px',
        width: '100%',
        '& .MuiExpansionPanelSummary-content': {
          margin: '0px',
          display: 'inline-block',
        },
        '&.Mui-expanded': {
          minHeight: '22px',
        },
        '& .MuiExpansionPanelSummary-expandIcon': {
          display: 'inline-block',
          padding: '0px',
          marginTop: '-1px',
        },
      },
      expansionPanelDetails: {
        padding: '12px',
        paddingTop: '0px',
        [theme.breakpoints.down('md')]: {
          display: 'block',
        },
      },
      intervalsRow: {
        borderBottom: '1px solid ' + colors.consoleStroke,
        width: '100%',
        lineHeight: '22px',
        '&:last-child': {
          borderBottom: 'none',
        },
      },
      intervalIcon: {
        display: 'inline-block',
        width: 25,
      },
      intervalStarted: {
        display: 'inline-block',
        width: 200,
      },
      intervalDuration: {
        display: 'inline-block',
      },
      intervalWarning: {
        display: 'inline-block',
        width: '100%',
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
          fontSize: '12px',
        },
      },
      button: {
        marginTop: 5,
        marginBottom: 10,
      },
      bottomSpace: {
        ...styles.bottomSpace,
      },
      artifactRow: {
        padding: '5px',
        cursor: 'pointer',
        [theme.breakpoints.down('sm')]: {
          paddingLeft: '0px',
          paddingRight: '0px',
          paddingTop: '10px',
        },
      },
      artifactName: {
        display: 'inline-block',
        width: '20%',
        [theme.breakpoints.down('sm')]: {
          display: 'block',
          width: '100%',
          marginBottom: '10px',
        },
        '& svg': {
          verticalAlign: 'middle',
        },
      },
      artifactDescription: {
        display: 'inline-block',
        width: '40%',
        [theme.breakpoints.down('sm')]: {
          display: 'block',
          width: '100%',
          marginBottom: '10px',
        },
      },
      artifactSize: {
        display: 'inline-block',
        width: '20%',
        [theme.breakpoints.down('sm')]: {
          display: 'block',
          width: '100%',
          marginBottom: '10px',
        },
      },
      artifactAction: {
        display: 'inline-block',
        width: '20%',
        textAlign: 'right',
        '& button': {
          marginBottom: '5px',
        },
        [theme.breakpoints.down('sm')]: {
          display: 'block',
          width: '100%',
        },
      },
      artifactExpansionPanel: {
        padding: '0px!important',
        boxShadow: 'none',
      },
      artifactExpansionPanelSummary: {
        display: 'none',
        minHeight: '0px!important',
      },
      artifactsExpansionPanelDetails: {
        padding: '0px!important',
      },
      summaryDivider: {
        minHeight: '10px',
      },
      rotate180Icon: {
        '& svg': {
          transform: 'rotate(180deg)',
        },
      },
      rotate0Icon: {
        '& svg': {
          transform: 'rotate(0deg)',
        },
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <DbLabSession {...props} classes={classes} />
}
