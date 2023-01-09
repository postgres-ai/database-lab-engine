/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useMemo } from 'react'
import { useStripe, useElements, CardElement } from '@stripe/react-stripe-js'
import { StripeCardElement } from '@stripe/stripe-js'
import { Button, makeStyles, Paper } from '@material-ui/core'

import { colors } from '@postgres.ai/shared/styles/colors'
import { icons } from '@postgres.ai/shared/styles/icons'

import Actions from '../actions/actions'

const useStyles = makeStyles(
  (theme) => ({
    yellowContainer: {
      background: 'yellow',
      color: 'red',
    },
    redContainer: {
      background: 'red',
      color: 'blue',
    },
    toolbar: theme.mixins.toolbar,
    cardContainer: {
      padding: 20,
      width: 500,
      display: 'inline-block',
      borderWidth: 1,
      borderColor: colors.consoleStroke,
      borderStyle: 'solid',
    },
    cardContainerTitle: {
      fontSize: 16,
      fontWeight: 'bold',
      marginTop: 0,
    },
    secureNotice: {
      fontSize: 10,
      float: 'right',
      height: 40,
      marginTop: -3,
      '& img': {
        float: 'right',
      },
    },
  }),
  { index: 1 },
)

const stripeStyles = (
  <style>
    {`
  label {
  /*color: #6b7c93;*/
  font-weight: bold;
  letter-spacing: 0.025em;
  font-size: 12px;
}

button {
  white-space: nowrap;
  border: 0;
  outline: 0;
  display: inline-block;
  height: 40px;
  line-height: 40px;
  padding: 0 14px;
  box-shadow: 0 4px 6px rgba(50, 50, 93, 0.11), 0 1px 3px rgba(0, 0, 0, 0.08);
  color: #fff;
  border-radius: 4px;
  font-size: 15px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.025em;
  background-color: #6772e5;
  text-decoration: none;
  -webkit-transition: all 150ms ease;
  transition: all 150ms ease;
  margin-top: 10px;
}

button:hover {
  color: #fff;
  cursor: pointer;
  background-color: #7795f8;
  transform: translateY(-1px);
  box-shadow: 0 7px 14px rgba(50, 50, 93, 0.1), 0 3px 6px rgba(0, 0, 0, 0.08);
}

input,
.StripeElement {
  display: block;
  margin: 10px 0 20px 0;
  max-width: 500px;
  padding: 10px 14px;
  font-size: 1em;
  font-family: "Source Code Pro", monospace;
  box-shadow: rgba(50, 50, 93, 0.14902) 0px 1px 3px,
    rgba(0, 0, 0, 0.0196078) 0px 1px 0px;
  border: 0;
  outline: 0;
  border-radius: 4px;
  background: white;
}

input::placeholder {
  color: #aab7c4;
}

input:focus,
.StripeElement--focus {
  box-shadow: rgba(50, 50, 93, 0.109804) 0px 4px 6px,
    rgba(0, 0, 0, 0.0784314) 0px 1px 3px;
  -webkit-transition: all 150ms ease;
  transition: all 150ms ease;
}

.StripeElement.IdealBankElement,
.StripeElement.FpxBankElement,
.StripeElement.PaymentRequestButton {
  padding: 0;
}

.StripeElement.PaymentRequestButton {
  height: 40px;
}
`}
  </style>
)

const useOptions = () => {
  const fontSize = '14px'
  const options = useMemo(
    () => ({
      hidePostalCode: true,
      style: {
        base: {
          fontSize,
          color: '#424770',
          letterSpacing: '0.025em',
          fontFamily: 'Source Code Pro, monospace',
          '::placeholder': {
            color: '#aab7c4',
          },
        },
        invalid: {
          color: '#9e2146',
        },
      },
    }),
    [fontSize],
  )

  return options
}

function StripeForm(props: {
  mode: string
  token: string | null
  orgId: number
  disabled: boolean
}) {
  const classes = useStyles()
  const stripe = useStripe()
  const elements = useElements()
  const options = useOptions()
  const subscriptionMode = props.mode

  const handleSubmit = async (event: { preventDefault: () => void }) => {
    event.preventDefault()

    if (!stripe || !elements) {
      // Stripe.js has not loaded yet. Make sure to disable
      // form submission until Stripe.js has loaded.
      return
    }

    const payload = await stripe.createPaymentMethod({
      type: 'card',
      card: elements.getElement(CardElement) as StripeCardElement,
    })
    console.log('[PaymentMethod]', payload)

    if (payload.error && payload.error.message) {
      Actions.setSubscriptionError(payload.error.message)
      return
    }

    Actions.subscribeBilling(
      props.token,
      props.orgId,
      payload.paymentMethod?.id,
    )
  }

  let buttonTitle = 'Subscribe'
  let messages = (
    <div>
      <p className={classes?.cardContainerTitle}>Enter payment details</p>
      <p>Your subscription will start now.</p>
    </div>
  )

  if (subscriptionMode === 'resume') {
    buttonTitle = 'Resume subscription'
    messages = (
      <div>
        <p className={classes?.cardContainerTitle}>Enter payment details</p>
        <p>Your subscription will be resumed now.</p>
      </div>
    )
  }

  if (subscriptionMode === 'update') {
    buttonTitle = 'Update subscription'
    messages = (
      <div>
        <p className={classes?.cardContainerTitle}>Update payment details</p>
        <p>Your payment details will be updated now.</p>
      </div>
    )
  }

  return (
    <div>
      <Paper className={classes?.cardContainer}>
        {messages}
        <label>
          {stripeStyles}
          Card details
          <CardElement options={options} />
        </label>

        <Button
          variant="contained"
          color="primary"
          disabled={!stripe || props.disabled}
          onClick={handleSubmit}
        >
          {buttonTitle}
        </Button>
        <div className={classes?.secureNotice}>{icons.poweredByStripe}</div>
      </Paper>
    </div>
  )
}

export default StripeForm
