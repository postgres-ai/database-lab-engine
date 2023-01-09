/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react'
import { NavLink } from 'react-router-dom'
import { loadStripe } from '@stripe/stripe-js'
import { Elements } from '@stripe/react-stripe-js'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Paper,
  ExpansionPanel,
  ExpansionPanelSummary,
  ExpansionPanelDetails,
} from '@material-ui/core'
import { Link } from '@postgres.ai/shared/components/Link2'
import ExpandMoreIcon from '@material-ui/icons/ExpandMore'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { icons } from '@postgres.ai/shared/styles/icons'
import { ClassesType } from '@postgres.ai/platform/src/components/types'

import ConsolePageTitle from '../ConsolePageTitle'
import StripeForm from '../StripeForm'
import settings from '../../utils/settings'
import format from '../../utils/format'
import Urls from '../../utils/urls'
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
  unsubscribe: () => void
  componentDidMount() {
    const that = this
    const { orgId } = this.props

    this.unsubscribe = Store.listen(function () {
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

  getDataUsageTable(isPriveleged: boolean) {
    const billing =
      this.state && this.state.data && this.state.data.billing
        ? this.state.data.billing
        : null
    const { orgId, classes } = this.props
    let unitAmount = 0
    let tibAmount = 0
    let periodAmount = 0
    let estAmount = 0
    let startDate
    let endDate
    let period

    if (
      !billing ||
      (billing && billing.isProcessing) ||
      (billing && billing.orgId !== orgId)
    ) {
      return <PageSpinner />
    }

    if (!billing) {
      return null
    }

    if (billing.data && billing.data.unit_amount) {
      // Anatoly: Logic behind `/100` is unknown, but currently we store 0.26 in the DB
      // So unitAmount will have the right value of 0.0026 per GiB*hour.
      unitAmount = parseFloat(billing.data.unit_amount) / 100
      tibAmount = unitAmount * 1024
    }

    if (billing && billing.data) {
      const periodDataUsage = parseFloat(billing.data.data_usage_sum)
      const periodEstDataUsage =
        parseFloat(billing.data.data_usage_estimate) + periodDataUsage
      if (!isPriveleged && periodDataUsage) {
        periodAmount = periodDataUsage * unitAmount
      }
      if (!isPriveleged && periodEstDataUsage) {
        estAmount = periodEstDataUsage * unitAmount
      }
      if (billing.data.period_start) {
        startDate = format.formatDate(billing.data.period_start)
      }
      if (billing.data.period_now) {
        endDate = format.formatDate(billing.data.period_now)
      }
    }

    if (!startDate && !endDate) {
      period = '-'
    } else {
      period = startDate + ' – ' + endDate
    }

    return (
      <>
        <>
          <Paper className={classes.paper}>
            <div className={classes.monthInfo}>
              <div className={classes.paperSection}>
                <strong>Current month</strong>
                <br />
                {period}
              </div>
              <div className={classes.paperSection}>
                <div className={classes.monthColumn}>
                  <strong>Month-to-date total cost</strong>&nbsp;&nbsp;
                  <Tooltip
                    title={
                      <React.Fragment>
                        Total cost for the {period} interval.
                      </React.Fragment>
                    }
                    classes={{ tooltip: classes.toolTip }}
                  >
                    {icons.infoIcon}
                  </Tooltip>
                  <br />
                  <strong className={classes.monthValue}>
                    ${this.toFixed(periodAmount)}
                  </strong>
                </div>

                <div className={classes.monthColumn}>
                  <strong>End-of-month total cost (forecast)</strong>
                  &nbsp;&nbsp;
                  <Tooltip
                    title={
                      <React.Fragment>
                        The forecast for this period is a sum of the actual cost
                        to the date and the projected cost based on average
                        usage from {period}.
                      </React.Fragment>
                    }
                    classes={{ tooltip: classes.toolTip }}
                  >
                    {icons.infoIcon}
                  </Tooltip>
                  <br />
                  <strong className={classes.monthValue}>
                    ${this.toFixed(estAmount)}
                  </strong>
                </div>
              </div>
              This is not an invoice
            </div>
          </Paper>

          <ExpansionPanel className={classes.expansionPaper}>
            <ExpansionPanelSummary
              expandIcon={<ExpandMoreIcon />}
              aria-controls="panel1a-content"
              id="panel1a-header"
              className={classes.expansionPaperHeader}
            >
              <strong>How is billing calculated?</strong>
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.expansionPaperBody}>
              <p>
                Billing is based on the total size of the databases running
                within Database Lab.
              </p>
              <p>
                The base cost per TiB per hour:&nbsp;
                <strong>${tibAmount && this.toFixed(tibAmount)}</strong>.<br />
                Discounts are not shown here and will be applied when the
                invoice is issued.
              </p>
              <p>
                We account only for the actual physical disk space used and
                monitor this hourly with 1 GiB precision. Free disk space is
                always ignored. The logical size of the database also does not
                factor into our calculation.
              </p>
              <Link to="https://postgres.ai/docs/pricing" target="_blank">
                Learn more
              </Link>
            </ExpansionPanelDetails>
          </ExpansionPanel>
        </>

        <h1>Data usage</h1>
        {billing.data &&
        billing.data.data_usage &&
        billing.data.data_usage.length ? (
          <HorizontalScrollContainer>
            <Table className={classes.table}>
              <TableHead>
                <TableRow className={classes.row}>
                  <TableCell>Database Lab instance ID</TableCell>
                  <TableCell>
                    Date&nbsp;
                    <span className={classes.sortArrow}>
                      {icons.sortArrowUp}
                    </span>
                  </TableCell>
                  <TableCell>Consumption, GiB·h</TableCell>
                  <TableCell>Amount, $</TableCell>
                  <TableCell>Billable</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {billing.data.data_usage.map((d) => {
                  return (
                    <TableRow hover className={classes.row} key={d.id}>
                      <TableCell className={classes.cell}>
                        <NavLink
                          to={Urls.linkDbLabInstance(this.props, d.instance_id)}
                          target="_blank"
                          className={classes.navLink}
                        >
                          {d.instance_id}
                        </NavLink>
                      </TableCell>
                      <TableCell className={classes.cell}>
                        {format.formatDate(d.day_date)}
                      </TableCell>
                      <TableCell className={classes.cell}>
                        {d.data_size_gib}
                      </TableCell>
                      <TableCell>
                        {!isPriveleged && d.to_invoice
                          ? this.toFixed(d.data_size_gib * unitAmount)
                          : 0}
                      </TableCell>
                      <TableCell className={classes.cell}>
                        {d.to_invoice ? 'Yes' : 'No'}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>
        ) : (
          'Data usage metrics are not gathered yet.'
        )}
      </>
    )
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
    const dataUsage = this.getDataUsageTable(orgData.is_priveleged)

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

    let subscription = null

    let mode = 'new'
    if (orgData.is_blocked && orgData.stripe_subscription_id) {
      mode = 'resume'
    }
    if (!orgData.is_blocked && orgData.stripe_subscription_id) {
      mode = 'update'
    }

    if (!orgData.is_priveleged) {
      subscription = (
        <div className={classes.subscriptionForm}>
          {orgData.stripe_subscription_id && (
            <div>
              {!orgData.is_blocked ? (
                <span>Subscription is active</span>
              ) : (
                <div className={classes.billingError}>
                  {icons.warningIcon}&nbsp;Subscription is NOT active.&nbsp;
                  {orgData.new_subscription
                    ? 'Payment processing.'
                    : 'Payment processing error.'}
                </div>
              )}
            </div>
          )}

          {!orgData.stripe_subscription_id && (
            <div className={classes.billingError}>
              {!orgData.is_blocked_on_creation ? (
                <div>
                  {icons.warningIcon}&nbsp; Trial period is expired. Enter
                  payment details to activate the organization.
                </div>
              ) : (
                <div>
                  {icons.warningIcon}&nbsp;Enter payment details to activate the
                  organization.
                </div>
              )}
            </div>
          )}

          <br />

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
      )
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {<ConsolePageTitle title={page.title} />}

        {orgData.is_blocked && !orgData.is_priveleged && (
          <span className={classes.orgStatusBlocked}>
            Organization is suspended.
          </span>
        )}

        {!orgData.is_blocked && orgData.is_priveleged && (
          <span className={classes.orgStatusActive}>
            Subscription is active till{' '}
            {format.formatTimestampUtc(orgData.priveleged_until)}.
          </span>
        )}

        {!orgData.is_blocked &&
          !orgData.is_priveleged &&
          orgData.stripe_subscription_id && (
            <span className={classes.orgStatusActive}>
              Subscription is active. Payment details are set.
            </span>
          )}

        {mode !== 'update' && subscription}

        {!this.props.short && dataUsage}

        <div className={classes.bottomSpace} />
      </div>
    )
  }
}

export default Billing
