import { makeStyles } from '@material-ui/core'
import Billing from 'components/Billing/Billing'
import { colors } from '@postgres.ai/shared/styles/colors'
import { styles } from '@postgres.ai/shared/styles/styles'

export interface BillingProps {
  org: string | number
  orgId: number
  short: boolean
  projectId: number | string | undefined
  orgData: {
    is_priveleged: boolean
    is_blocked: boolean
    new_subscription: boolean
    is_blocked_on_creation: boolean
    stripe_subscription_id: number
    priveleged_until: Date
    role: {
      id: number
    }
  }
}

export const BillingWrapper = (props: BillingProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        '& ul': {
          '& > li': {
            'list-style-position': 'inside',
          },
          padding: 0,
        },
        '& h1': {
          fontSize: '16px!important',
          fontWeight: 'bold',
        },
        '& h2': {
          fontSize: '14px!important',
          fontWeight: 'bold',
        },
        width: '100%',
        'min-height': '100%',
        'z-index': 1,
        position: 'relative',
        [theme.breakpoints.down('sm')]: {
          maxWidth: '100vw',
        },
        [theme.breakpoints.up('md')]: {
          maxWidth: 'calc(100vw - 200px)',
        },
        [theme.breakpoints.up('lg')]: {
          maxWidth: 'calc(100vw - 200px)',
        },
        'font-size': '14px!important',
        'font-family': '"Roboto", "Helvetica", "Arial", sans-serif',

        display: 'flex',
        flexDirection: 'column',
        paddingBottom: '20px',
      },
      billingError: {
        color: colors.state.warning,
      },
      errorMessage: {
        color: colors.state.error,
        marginBottom: 10,
      },
      subscriptionForm: {
        marginBottom: 20,
      },
      orgStatusActive: {
        color: colors.state.ok,
        display: 'block',
        marginBottom: 20,
      },
      orgStatusBlocked: {
        color: colors.state.error,
        display: 'block',
        marginBottom: 20,
      },
      navLink: {
        color: colors.secondary2.main,
        '&:visited': {
          color: colors.secondary2.main,
        },
      },
      sortArrow: {
        '& svg': {
          marginBottom: -8,
        },
      },
      paperSection: {
        display: 'block',
        width: '100%',
        marginBottom: 20,
        overflow: 'auto',
      },
      monthColumn: {
        width: 255,
        float: 'left',
      },
      monthInfo: {
        '& strong': {
          display: 'inline-block',
          marginBottom: 10,
        },
      },
      monthValue: {
        marginBottom: '0px!important',
      },

      toolTip: {
        fontSize: '12px!important',
        maxWidth: '300px!important',
      },
      paper: {
        maxWidth: 510,
        padding: 15,
        marginBottom: 20,
        display: 'block',
        borderWidth: 1,
        borderColor: colors.consoleStroke,
        borderStyle: 'solid',
      },
      expansionPaper: {
        maxWidth: 540,
        borderWidth: 1,
        borderColor: colors.consoleStroke,
        borderStyle: 'solid',
        borderRadius: 4,
        marginBottom: 30,
      },
      expansionPaperHeader: {
        padding: 15,
        minHeight: 0,
        'justify-content': 'left',
        '& div.MuiExpansionPanelSummary-content': {
          margin: 0,
        },
        '&.Mui-expanded': {
          minHeight: '0px!important',
        },
        '& .MuiExpansionPanelSummary-expandIcon': {
          padding: 0,
          marginRight: 0,
        },
      },
      expansionPaperBody: {
        padding: 15,
        paddingTop: 0,
        display: 'block',
        marginTop: -15,
      },
      bottomSpace: {
        ...styles.bottomSpace,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <Billing {...props} classes={classes} />
}
