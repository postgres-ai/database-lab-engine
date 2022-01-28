/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { NavLink } from 'react-router-dom';
import { withStyles } from '@material-ui/core/styles';
import { loadStripe } from '@stripe/stripe-js';
import { Elements } from '@stripe/react-stripe-js';
import {
  Table, TableBody, TableCell,
  TableHead, TableRow, Tooltip, Paper
} from '@material-ui/core';
import Link from './Link';
import ExpansionPanel from '@material-ui/core/ExpansionPanel';
import ExpansionPanelSummary from '@material-ui/core/ExpansionPanelSummary';
import ExpansionPanelDetails from '@material-ui/core/ExpansionPanelDetails';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { colors } from '@postgres.ai/shared/styles/colors';
import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';

import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import StripeForm from './StripeForm';
import settings from '../utils/settings';
import format from '../utils/format';
import Urls from '../utils/urls';
import Store from '../stores/store';
import Actions from '../actions/actions';
import Permissions from '../utils/permissions.js';
import Error from './Error';

const stripePromise = loadStripe(settings.stripeApiKey, {
  locale: 'en'
});


const getStyles = theme => ({
  root: {
    '& ul': {
      '& > li': {
        'list-style-position': 'inside'
      },
      'padding': 0
    },
    '& h1': {
      fontSize: '16px!important',
      fontWeight: 'bold'
    },
    '& h2': {
      fontSize: '14px!important',
      fontWeight: 'bold'
    },
    'width': '100%',
    'min-height': '100%',
    'z-index': 1,
    'position': 'relative',
    [theme.breakpoints.down('sm')]: {
      maxWidth: '100vw'
    },
    [theme.breakpoints.up('md')]: {
      maxWidth: 'calc(100vw - 200px)'
    },
    [theme.breakpoints.up('lg')]: {
      maxWidth: 'calc(100vw - 200px)'
    },
    'font-size': '14px!important',
    'font-family': '"Roboto", "Helvetica", "Arial", sans-serif',

    'display': 'flex',
    'flexDirection': 'column',
    'paddingBottom': '20px'
  },
  billingError: {
    color: colors.state.warning
  },
  errorMessage: {
    color: colors.state.error,
    marginBottom: 10
  },
  subscriptionForm: {
    marginBottom: 20
  },
  orgStatusActive: {
    color: colors.state.ok,
    display: 'block',
    marginBottom: 20
  },
  orgStatusBlocked: {
    color: colors.state.error,
    display: 'block',
    marginBottom: 20
  },
  navLink: {
    'color': colors.secondary2.main,
    '&:visited': {
      color: colors.secondary2.main
    }
  },
  sortArrow: {
    '& svg': {
      marginBottom: -8
    }
  },
  paperSection: {
    display: 'block',
    width: '100%',
    marginBottom: 20,
    overflow: 'auto'
  },
  monthColumn: {
    width: 255,
    float: 'left'
  },
  monthInfo: {
    '& strong': {
      display: 'inline-block',
      marginBottom: 10
    }
  },
  monthValue: {
    marginBottom: '0px!important'
  },

  toolTip: {
    fontSize: '12px!important',
    maxWidth: '300px!important'
  },
  paper: {
    maxWidth: 510,
    padding: 15,
    marginBottom: 20,
    display: 'block',
    borderWidth: 1,
    borderColor: colors.consoleStroke,
    borderStyle: 'solid'
  },
  expansionPaper: {
    maxWidth: 540,
    borderWidth: 1,
    borderColor: colors.consoleStroke,
    borderStyle: 'solid',
    borderRadius: 4,
    marginBottom: 30
  },
  expansionPaperHeader: {
    'padding': 15,
    'minHeight': 0,
    'justify-content': 'left',
    '& div.MuiExpansionPanelSummary-content': {
      margin: 0
    },
    '&.Mui-expanded': {
      minHeight: '0px!important'
    },
    '& .MuiExpansionPanelSummary-expandIcon': {
      padding: 0,
      marginRight: 0
    }
  },
  expansionPaperBody: {
    padding: 15,
    paddingTop: 0,
    display: 'block',
    marginTop: -15
  },
  bottomSpace: {
    ...styles.bottomSpace
  }
});

const page = {
  title: 'Billing'
};

class Billing extends Component {
  componentDidMount() {
    const that = this;
    const { orgId } = this.props;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const billing = this.data && this.data.billing ? this.data.billing : null;

      that.setState({ data: this.data });

      if (auth && auth.token && billing &&
        !billing.isProcessing && !billing.error && !that.state) {
        Actions.getBillingDataUsage(auth.token, orgId);
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  toFixed(value) {
    if (value && value.toFixed && value !== 0) {
      return value.toFixed(4);
    }

    return '0.0';
  }

  getDataUsageTable(isPriveleged) {
    const billing = this.state && this.state.data &&
      this.state.data.billing ?
      this.state.data.billing : null;
    const { orgId, classes } = this.props;
    let unitAmount;
    let tibAmount;
    let periodAmount = 0;
    let estAmount = 0;
    let startDate;
    let endDate;
    let period;

    if (!billing || (billing && billing.isProcessing) || (billing && billing.orgId !== orgId)) {
      return (
        <PageSpinner />
      );
    }

    if (!billing) {
      return null;
    }

    if (billing.data && billing.data.unit_amount) {
      // Anatoly: Logic behind `/100` is unknown, but currently we store 0.26 in the DB
      // So unitAmount will have the right value of 0.0026 per GiB*hour.
      unitAmount = (parseFloat(billing.data.unit_amount) / 100);
      tibAmount = (unitAmount * 1024);
    }

    if (billing && billing.data) {
      let periodDataUsage = parseFloat(billing.data.data_usage_sum);
      let periodEstDataUsage = parseFloat(billing.data.data_usage_estimate) + periodDataUsage;
      if (!isPriveleged && periodDataUsage) {
        periodAmount = periodDataUsage * unitAmount;
      }
      if (!isPriveleged && periodEstDataUsage) {
        estAmount = periodEstDataUsage * unitAmount;
      }
      if (billing.data.period_start) {
        startDate = format.formatDate(billing.data.period_start);
      }
      if (billing.data.period_now) {
        endDate = format.formatDate(billing.data.period_now);
      }
    }

    if (!startDate && !endDate) {
      period = '-';
    } else {
      period = startDate + ' – ' + endDate;
    }

    return (
      <>
        <>
          <Paper className={classes.paper}>
            <div className={classes.monthInfo}>
              <div className={classes.paperSection}>
                <strong>Current month</strong>
                <br/>
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
                  <br/>
                  <strong className={classes.monthValue}>
                    ${this.toFixed(periodAmount)}
                  </strong>
                </div>

                <div className={classes.monthColumn}>
                  <strong>End-of-month total cost (forecast)</strong>&nbsp;&nbsp;
                  <Tooltip
                    title={
                      <React.Fragment>
                        The forecast for this period is a sum
                        of the actual cost to the date and the projected
                        cost based on average usage from {period}.
                      </React.Fragment>
                    }
                    classes={{ tooltip: classes.toolTip }}
                  >
                    {icons.infoIcon}
                  </Tooltip><br/>
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
              aria-controls='panel1a-content'
              id='panel1a-header'
              className={classes.expansionPaperHeader}
            >
              <strong>How is billing calculated?</strong>
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.expansionPaperBody}>
              <p>Billing is based on the total size of the databases
                running within Database Lab.</p>
              <p>The base cost per TiB per hour:&nbsp;
                <strong>
                  ${tibAmount && this.toFixed(tibAmount)}
                </strong>.<br/>
                Discounts are not shown here and will be applied when the
                invoice is issued.</p>
              <p>We account only for the actual physical disk space used and monitor
                this hourly with 1 GiB precision. Free disk space is always ignored.
                The logical size of the database also does not factor into our
                calculation.
              </p>
              <Link
                link='https://postgres.ai/docs/pricing'
                target='_blank'
              >
                Learn more
              </Link>
            </ExpansionPanelDetails>
          </ExpansionPanel>
        </>

        <h1>Data usage</h1>
        {billing.data && billing.data.data_usage && billing.data.data_usage.length ? (
          <HorizontalScrollContainer>
            <Table className={classes.table}>
              <TableHead>
                <TableRow className={classes.row}>
                  <TableCell>Database Lab instance ID</TableCell>
                  <TableCell>
                    Date&nbsp;
                    <span className={classes.sortArrow}>{icons.sortArrowUp}</span>
                  </TableCell>
                  <TableCell>Consumption, GiB·h</TableCell>
                  <TableCell>Amount, $</TableCell>
                  <TableCell>Billable</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {billing.data.data_usage.map(d => {
                  return (
                    <TableRow
                      hover
                      className={classes.row}
                      key={d.id}
                    >
                      <TableCell className={classes.cell}>
                        <NavLink
                          to={Urls.linkDbLabInstance(this.props, d.instance_id)}
                          target='_blank'
                          className={classes.navLink}
                        >
                          {d.instance_id}
                        </NavLink>
                      </TableCell>
                      <TableCell className={classes.cell}>
                        {format.formatDate(d.day_date)}
                      </TableCell>
                      <TableCell className={classes.cell}>{d.data_size_gib}</TableCell>
                      <TableCell>
                        {!isPriveleged && d.to_invoice ?
                          this.toFixed(d.data_size_gib * unitAmount) : 0}
                      </TableCell>
                      <TableCell className={classes.cell}>
                        {d.to_invoice ? 'Yes' : 'No'}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>) : 'Data usage metrics are not gathered yet.'}
      </>
    );
  }

  render() {
    const { classes, orgId } = this.props;
    const auth = this.state && this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const data = this.state && this.state.data && this.state.data.billing ?
      this.state.data.billing : null;
    const orgData = this.props.orgData;
    const dataUsage = this.getDataUsageTable(orgData.is_priveleged);

    const breadcrumbs = (
      <ConsoleBreadcrumbs
        {...this.props}
        breadcrumbs={[
          { name: page.title }
        ]}
      />
    );

    if (!Permissions.isAdmin(orgData)) {
      return (
        <div>
          {breadcrumbs}

          {<ConsolePageTitle
            title={page.title}
          />}

          <Error
            message='You do not have permission to view this page.'
            code={403}
          />

        </div>
      );
    }

    let subscription = null;

    let mode = 'new';
    if (orgData.is_blocked && orgData.stripe_subscription_id) {
      mode = 'resume';
    }
    if (!orgData.is_blocked && orgData.stripe_subscription_id) {
      mode = 'update';
    }

    if (!orgData.is_priveleged) {
      subscription = (
        <div className={classes.subscriptionForm}>
          {orgData.stripe_subscription_id && (
            <div>
              {!orgData.is_blocked ?
                (<span>Subscription is active</span>) :
                (
                  <div className={classes.billingError}>
                    {icons.warningIcon}&nbsp;Subscription is NOT active.&nbsp;
                    {orgData.new_subscription ?
                      'Payment processing.' : 'Payment processing error.'}
                  </div>
                )
              }
            </div>
          )}

          {!orgData.stripe_subscription_id && (
            <div className={classes.billingError}>
              {!orgData.is_blocked_on_creation ? (
                <div>
                  {icons.warningIcon}&nbsp;
                  Trial period is expired. Enter payment details to activate the organization.
                </div>
              ) : (
                <div>
                  {icons.warningIcon}&nbsp;Enter payment details to activate the organization.
                </div>
              )}
            </div>
          )}

          <br/>

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
                    disabled={data && data.isSubscriptionProcessing}
                  />
                </Elements>
              </div>
            )}
          </div>
        </div>
      );
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {<ConsolePageTitle
          title={page.title}
        />}

        {orgData.is_blocked && !orgData.is_priveleged && (
          <span className={classes.orgStatusBlocked}>Organization is suspended.</span>
        )}

        {!orgData.is_blocked && orgData.is_priveleged && (
          <span className={classes.orgStatusActive}>
            Subscription is active till {format.formatTimestampUtc(orgData.priveleged_until)}.
          </span>
        )}

        {!orgData.is_blocked && !orgData.is_priveleged && orgData.stripe_subscription_id && (
          <span className={classes.orgStatusActive}>
            Subscription is active. Payment details are set.
          </span>
        )}

        {mode !== 'update' && subscription}

        {!this.props.short && dataUsage }

        <div className={classes.bottomSpace}/>
      </div>
    );
  }
}

Billing.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(Billing);
