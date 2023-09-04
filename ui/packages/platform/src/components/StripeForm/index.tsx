/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */
import { Box } from '@mui/material'
import { formatDistanceToNowStrict } from 'date-fns'
import { useReducer, useEffect, useCallback, ReducerAction } from 'react'
import { Button, makeStyles, Paper, Tooltip } from '@material-ui/core'

import { colors } from '@postgres.ai/shared/styles/colors'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { stripeStyles } from 'components/StripeForm/stripeStyles'
import { Link } from '@postgres.ai/shared/components/Link2'
import { ROUTES } from 'config/routes'

import { getPaymentMethods } from 'api/billing/getPaymentMethods'
import { startBillingSession } from 'api/billing/startBillingSession'
import { getSubscription } from 'api/billing/getSubscription'

import format from '../../utils/format'

interface BillingSubscription {
  status: string
  created_at: number
  description: string
  plan_description: string
  recognized_dblab_instance_id: number
  selfassigned_instance_id: string
  subscription_id: string
  telemetry_last_reported: number
  telemetry_usage_total_hours_last_3_months: string
}

interface HourEntry {
  [key: string]: number
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const useStyles = makeStyles(
  (theme) => ({
    paperContainer: {
      display: 'flex',
      flexDirection: 'row',
      alignItems: 'flex-start',
      gap: 20,
      width: '100%',
      height: '100%',

      [theme.breakpoints.down('sm')]: {
        flexDirection: 'column',
      },
    },
    cardContainer: {
      padding: 20,
      display: 'inline-block',
      borderWidth: 1,
      borderColor: colors.consoleStroke,
      borderStyle: 'solid',
      flex: '1 1 0',

      [theme.breakpoints.down('sm')]: {
        width: '100%',
        flex: 'auto',
      },
    },
    subscriptionContainer: {
      display: 'flex',
      flexDirection: 'column',
      flex: '1 1 0',
      gap: 20,

      [theme.breakpoints.down('sm')]: {
        width: '100%',
        flex: 'auto',
      },
    },
    flexColumnContainer: {
      display: 'flex',
      flexDirection: 'column',
      flex: '1.75 1 0',
      gap: 20,

      [theme.breakpoints.down('sm')]: {
        width: '100%',
        flex: 'auto',
      },
    },
    cardContainerTitle: {
      fontSize: 14,
      fontWeight: 'bold',
      margin: 0,
    },
    label: {
      fontSize: 14,
    },
    saveButton: {
      marginTop: 15,
      display: 'flex',
    },
    error: {
      color: '#E42548',
      fontSize: 12,
      marginTop: '-10px',
    },
    checkboxRoot: {
      fontSize: 14,
      marginTop: 10,
    },
    spinner: {
      marginLeft: '8px',
    },
    emptyMethod: {
      fontSize: 13,
      flex: '1 1 0',
    },
    button: {
      '&:hover': {
        color: '#0F879D',
      },
    },
    columnContainer: {
      display: 'flex',
      alignItems: 'center',
      flexDirection: 'row',
      padding: '10px 0',
      borderBottom: '1px solid rgba(224, 224, 224, 1)',

      '& p:first-child': {
        flex: '1 1 0',
        fontWeight: '500',
      },

      '& p': {
        flex: '2 1 0',
        margin: '0',
        fontSize: 13,
      },

      '&:last-child': {
        borderBottom: 'none',
      },
    },
    toolTip: {
      fontSize: '10px !important',
      maxWidth: '100%',
    },
    timeLabel: {
      lineHeight: '16px',
      fontSize: 13,
      cursor: 'pointer',
      flex: '2 1 0',
    },
    card: {
      display: 'flex',
      flexDirection: 'row',
      alignItems: 'center',
      gap: 20,
      position: 'relative',
      justifyContent: 'space-between',
      marginTop: 20,
      minHeight: 45,

      '& img': {
        objectFit: 'contain',
      },
    },
  }),
  { index: 1 },
)

function StripeForm(props: {
  alias: string
  mode: string
  token: string | null
  orgId: number
  disabled: boolean
}) {
  const classes = useStyles()

  const initialState = {
    isLoading: false,
    isFetching: false,
    cards: [],
    billingInfo: [],
  }

  const reducer = (
    state: typeof initialState,
    // @ts-ignore
    action: ReducerAction<unknown, void>,
  ) => {
    switch (action.type) {
      case 'setIsLoading':
        return { ...state, isLoading: action.isLoading }
      case 'setIsFetching':
        return { ...state, isFetching: action.isFetching }
      case 'setBillingInfo':
        return { ...state, billingInfo: action.billingInfo, isFetching: false }
      case 'setCards':
        const updatedCards = state.cards.concat(action.cards)
        const uniqueCards = updatedCards.filter(
          (card: { id: string }, index: number) =>
            updatedCards.findIndex((c: { id: string }) => c.id === card.id) ===
            index,
        )

        return {
          ...state,
          cards: uniqueCards,
          isLoading: false,
        }
      default:
        throw new Error()
    }
  }

  const [state, dispatch] = useReducer(reducer, initialState)

  const handleSubmit = async (event: { preventDefault: () => void }) => {
    event.preventDefault()

    dispatch({
      type: 'setIsLoading',
      isLoading: true,
    })

    startBillingSession(props.orgId, window.location.href).then((res) => {
      dispatch({
        type: 'setIsLoading',
        isLoading: false,
      })

      if (res.response) {
        window.open(res.response.details.content.url, '_blank')
      }
    })
  }

  const fetchPaymentMethods = useCallback(() => {
    dispatch({
      type: 'setIsFetching',
      isFetching: true,
    })

    getPaymentMethods(props.orgId).then((res) => {
      dispatch({
        type: 'setCards',
        cards: res.response.details.content.data,
      })

      getSubscription(props.orgId).then((res) => {
        dispatch({
          type: 'setBillingInfo',
          billingInfo: res.response.summary,
        })
      })
    })
  }, [props.orgId])

  useEffect(() => {
    fetchPaymentMethods()

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        fetchPaymentMethods()
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange, false)

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [props.orgId, fetchPaymentMethods])

  if (state.isFetching) {
    return (
      <div className={classes.paperContainer}>
        <Spinner />
      </div>
    )
  }

  const BillingDetail = ({
    label,
    value,
    isDateValue,
    isLink,
    instanceId,
  }: {
    label: string
    value: string | number
    isDateValue?: boolean
    isLink?: boolean
    instanceId?: string | number
  }) => (
    <Box className={classes.columnContainer}>
      <p className={classes.emptyMethod}>{label}</p>
      {isDateValue && value ? (
        <span className={classes.timeLabel}>
          <Tooltip
            title={format.formatTimestampUtc(value) ?? ''}
            classes={{ tooltip: classes.toolTip }}
          >
            <span>
              {formatDistanceToNowStrict(new Date(value), {
                addSuffix: true,
              })}
            </span>
          </Tooltip>
        </span>
      ) : isLink && value ? (
        <p>
          <Link
            to={ROUTES.ORG.INSTANCES.INSTANCE.createPath({
              org: props.alias,
              instanceId: String(instanceId),
            })}
            className={classes.emptyMethod}
          >
            {instanceId}
          </Link>
          &nbsp;
          {`(self-assigned: ${value})` || '---'}
        </p>
      ) : (
        <p
          className={classes.emptyMethod}
          style={
            label === 'Subscription status'
              ? { textTransform: 'capitalize' }
              : {}
          }
        >
          {value || '---'}
        </p>
      )}
    </Box>
  )

  const formatHoursUsed = (hours: HourEntry[] | string) => {
    if (typeof hours === 'string' || !hours) {
      return 'N/A'
    }

    const formattedHours = hours.map((entry: HourEntry) => {
      const key = Object.keys(entry)[0]
      const value = entry[key]
      return `${key}: ${value}`
    })

    return formattedHours.join('\n')
  }

  return (
    <div>
      {stripeStyles}
      <div className={classes.paperContainer}>
        <Box className={classes.subscriptionContainer}>
          <p className={classes?.cardContainerTitle}>Subscriptions</p>
          {state.billingInfo.length > 0 ? (
            state.billingInfo.map(
              (billing: BillingSubscription, index: number) => (
                <Paper className={classes?.cardContainer} key={index}>
                  <BillingDetail
                    label="Plan"
                    value={billing?.plan_description}
                  />
                  <BillingDetail
                    label="Subscription status"
                    value={billing.status}
                  />
                  <BillingDetail
                    label="Subscription ID"
                    value={billing?.subscription_id}
                  />
                  <BillingDetail
                    isLink={!!billing?.recognized_dblab_instance_id}
                    label="Instance ID"
                    instanceId={billing?.recognized_dblab_instance_id || 'N/A'}
                    value={billing?.selfassigned_instance_id}
                  />
                  <BillingDetail
                    isDateValue
                    label="Registered"
                    value={billing?.created_at}
                  />
                  <BillingDetail
                    isDateValue
                    label="Telemetry last reported"
                    value={billing?.telemetry_last_reported}
                  />
                  <BillingDetail
                    label="Description"
                    value={billing?.description}
                  />
                  <BillingDetail
                    label="Hours billed (current & last 2 months)"
                    value={formatHoursUsed(
                      billing?.telemetry_usage_total_hours_last_3_months,
                    )}
                  />
                </Paper>
              ),
            )
          ) : (
            <Paper className={classes?.cardContainer}>
              <BillingDetail label="Plan" value={'---'} />
              <BillingDetail label="Subscription status" value={'---'} />
              <BillingDetail label="Subscription ID" value={'---'} />
              <BillingDetail label="Instance ID" value={'---'} />
              <BillingDetail label="Registered" value={'---'} />
              <BillingDetail label="Telemetry last reported" value={'---'} />
              <BillingDetail label="Description" value={'---'} />
              <BillingDetail
                label="Hours billed (current & last 2 months)"
                value={'---'}
              />
            </Paper>
          )}
        </Box>
        <Box className={classes.subscriptionContainer}>
          <p className={classes?.cardContainerTitle}>Payment methods</p>
          <Paper className={classes?.cardContainer}>
            <Box
              sx={{
                display: 'flex',
                justifyContent: 'flex-end',
                alignItems: 'center',
              }}
            >
              <Button
                variant="contained"
                color="primary"
                disabled={props.disabled || state.isLoading}
                onClick={handleSubmit}
              >
                Edit payment methods
                {state.isLoading && (
                  <Spinner size="sm" className={classes.spinner} />
                )}
              </Button>
              {state.cards.length > 0 && (
                <Button
                  variant="outlined"
                  color="secondary"
                  className={classes.button}
                  disabled={props.disabled || state.isLoading}
                  onClick={handleSubmit}
                >
                  Invoices
                  {state.isLoading && (
                    <Spinner size="sm" className={classes.spinner} />
                  )}
                </Button>
              )}
            </Box>
            {state.cards.length === 0 && !state.isFetching ? (
              <p className={classes.emptyMethod}>No payment methods available</p>
            ) : (
              <>
                {state.cards.map(
                  (
                    card: {
                      id: string
                      card: {
                        exp_year: string
                        exp_month: string
                        brand: string
                        last4: string
                      }
                    },
                    index: number,
                  ) => (
                    <Box className={classes.card} key={index}>
                      <Box
                        sx={{
                          display: 'flex',
                          justifyContent: 'center',
                          alignItems: 'center',
                          gap: '20px',

                          '& p': {
                            margin: '10px 0',
                          },
                        }}
                      >
                        <img
                          src={`/images/paymentMethods/${card.card.brand}.png`}
                          alt={card.card.brand}
                          width="50px"
                          height="35px"
                        />
                        <Box
                          sx={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            flexDirection: 'column',
                            alignItems: 'flex-start',
                            gap: '4px',
                            width: '100px',

                            '& > p': {
                              margin: 0,
                            },
                          }}
                        >
                          <p className={classes.emptyMethod}>
                            **** {card.card.last4}
                          </p>
                          <p className={classes.emptyMethod}>
                            Expires {card.card.exp_month}/{card.card.exp_year}
                          </p>
                        </Box>
                      </Box>
                    </Box>
                  ),
                )}
              </>
            )}
          </Paper>
        </Box>
      </div>
    </div>
  )
}

export default StripeForm
