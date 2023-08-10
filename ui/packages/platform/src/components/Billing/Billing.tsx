/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import { loadStripe } from '@stripe/stripe-js'
import { Elements } from '@stripe/react-stripe-js'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import ConsolePageTitle from '../ConsolePageTitle'
import StripeForm from '../StripeForm'
import settings from '../../utils/settings'
import Store from '../../stores/store'
import Actions from '../../actions/actions'
import Permissions from '../../utils/permissions'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { BillingProps } from 'components/Billing/BillingWrapper'

interface BillingWithStylesProps extends BillingProps {
  classes: ClassesType
}

interface BillingState {
  data: {
    auth: {
      token: string
    } | null
    billing: {
      orgId: number
      error: boolean
      isProcessing: boolean
      subscriptionError: boolean
      subscriptionErrorMessage: string
      isSubscriptionProcessing: boolean
      primaryPaymentMethod: string
      data: {
        unit_amount: string
        data_usage_estimate: string
        data_usage_sum: string
        data_usage: {
          id: number
          instance_id: string
          day_date: Date
          data_size_gib: number
          to_invoice: boolean
        }[]
        period_start: Date
        period_now: Date
      }
    }
  }
}

const stripePromise = loadStripe(settings.stripeApiKey as string, {
  locale: 'en',
})

const page = {
  title: 'Billing',
}

class Billing extends Component<BillingWithStylesProps, BillingState> {
  unsubscribe: Function
  componentDidMount() {
    const that = this
    const { orgId } = this.props

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      const auth: BillingState['data']['auth'] =
        this.data && this.data.auth ? this.data.auth : null
      const billing: BillingState['data']['billing'] =
        this.data && this.data.billing ? this.data.billing : null

      that.setState({ data: this.data })

      if (
        auth &&
        auth.token &&
        billing &&
        !billing.isProcessing &&
        !billing.error &&
        !that.state
      ) {
        Actions.getBillingDataUsage(auth.token, orgId)
      }
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  toFixed(value: number) {
    if (value && value.toFixed && value !== 0) {
      return value.toFixed(4)
    }

    return '0.0'
  }

  render() {
    const { classes, orgId, orgData } = this.props
    const auth =
      this.state && this.state.data && this.state.data.auth
        ? this.state.data.auth
        : null
    const data =
      this.state && this.state.data && this.state.data.billing
        ? this.state.data.billing
        : null

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        {...this.props}
        breadcrumbs={[{ name: page.title }]}
      />
    )

    if (!Permissions.isAdmin(orgData)) {
      return (
        <div>
          {breadcrumbs}

          {<ConsolePageTitle title={page.title} />}

          <ErrorWrapper
            message="You do not have permission to view this page."
            code={403}
          />
        </div>
      )
    }

    let mode = 'new'
    if (orgData.is_blocked && orgData.stripe_subscription_id) {
      mode = 'resume'
    }
    if (!orgData.is_blocked && orgData.stripe_subscription_id) {
      mode = 'update'
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}
        <ConsolePageTitle title={page.title} />
        <div className={classes.subscriptionForm}>
          <div>
            {Permissions.isAdmin(orgData) && (
              <div>
                {data && data.subscriptionError && (
                  <div className={classes.errorMessage}>
                    {data.subscriptionErrorMessage}
                  </div>
                )}
                <Elements stripe={stripePromise}>
                  <StripeForm
                    alias={orgData.alias}
                    mode={mode}
                    token={auth && auth.token ? auth.token : null}
                    orgId={orgId}
                    disabled={data !== null && data.isSubscriptionProcessing}
                  />
                </Elements>
              </div>
            )}
          </div>
        </div>
        <div className={classes.bottomSpace} />
      </div>
    )
  }
}

export default Billing
