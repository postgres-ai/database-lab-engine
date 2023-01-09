import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { OrgPermissions } from 'components/types'
import Audit from 'components/Audit/Audit'

export interface AuditProps {
  orgId: number
  org: string | number
  project: string | undefined
  orgPermissions: OrgPermissions
}

export const AuditWrapper = (props: AuditProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        ...(styles.root as Object),
        display: 'flex',
        flexDirection: 'column',
        paddingBottom: '20px',
      },
      container: {
        display: 'flex',
        flexWrap: 'wrap',
      },
      timeCell: {
        verticalAlign: 'top',
        minWidth: 200,
      },
      expansionPanel: {
        boxShadow: 'none',
        background: 'transparent',
        fontSize: '12px',
        marginBottom: '5px',
      },
      expansionPanelSummary: {
        display: 'inline-block',
        padding: '0px',
        minHeight: '22px',
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
        },
      },
      expansionPanelDetails: {
        padding: '0px',
        [theme.breakpoints.down('md')]: {
          display: 'block',
        },
      },
      actionDescription: {
        marginBottom: '5px',
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
      showMoreContainer: {
        marginTop: 20,
        textAlign: 'center',
      },
      data: {
        width: '50%',
        [theme.breakpoints.up('md')]: {
          width: '50%',
          marginRight: '10px',
        },
      },
      bottomSpace: {
        ...styles.bottomSpace,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <Audit {...props} classes={classes} />
}
