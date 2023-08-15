import { makeStyles } from '@material-ui/core'
import Billing from 'components/Billing/Billing'
import { colors } from '@postgres.ai/shared/styles/colors'

export interface BillingProps {
  org: string | number
  orgId: number
  short: boolean
  projectId: number | string | undefined
  orgData: {
    alias: string
    is_priveleged: boolean
    stripe_payment_method_primary: string
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
      errorMessage: {
        color: colors.state.error,
        marginBottom: 10,
      },
      subscriptionForm: {
        marginBottom: 20,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <Billing {...props} classes={classes} />
}
