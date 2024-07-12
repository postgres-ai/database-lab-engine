/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Button, makeStyles } from '@material-ui/core'

import { useCloudProviderProps } from 'hooks/useCloudProvider'
import { cloudProviderName, pricingPageForProvider } from './utils'

const MONTHLY_HOURS = 730

const useStyles = makeStyles({
  aside: {
    width: '100%',
    height: 'fit-content',
    minHeight: '300px',
    padding: '24px',
    borderRadius: '4px',
    boxShadow: '0 8px 16px #3a3a441f, 0 16px 32px #5a5b6a1f',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'flex-start',
    flex: '1 1 0',
    position: 'sticky',
    top: 10,

    '& > h2': {
      fontSize: '14px',
      fontWeight: 500,
      margin: '0 0 10px 0',
      height: 'fit-content',
    },

    '& > span': {
      fontSize: '13px',
    },

    '& > button': {
      padding: '10px 20px',
      marginTop: '20px',
    },

    '@media (max-width: 1200px)': {
      position: 'relative',
      boxShadow: 'none',
      borderRadius: '0',
      padding: '0',
      flex: 'auto',
      marginBottom: '30px',

      '& > button': {
        width: 'max-content',
      },
    },
  },
  asideSection: {
    padding: '12px 0',
    borderBottom: '1px solid #e0e0e0',

    '& > span': {
      color: '#808080',
    },

    '& > p': {
      margin: '5px 0 0 0',
      fontSize: '13px',
    },
  },
  flexWrap: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: '5px',
  },
  capitalize: {
    textTransform: 'capitalize',
  },
  remark: {
    fontSize: '11px',
    marginTop: '5px',
    display: 'block',
    lineHeight: '1.2',
  },
})

export const DbLabInstanceFormSidebar = ({
  cluster,
  state,
  handleCreate,
  disabled,
}: {
  cluster?: boolean
  state: useCloudProviderProps['initialState']
  handleCreate: () => void
  disabled: boolean
}) => {
  const classes = useStyles()

  return (
    <div className={classes.aside}>
      <h2>Preview</h2>
      <span>
        Review the specifications of the virtual machine, storage, and software
        that will be provisioned.
      </span>
      {state.name && (
        <div className={classes.asideSection}>
          <span>Name</span>
          <p>{state.name}</p>
        </div>
      )}
      {state.tag && (
        <div className={classes.asideSection}>
          <span>Tag</span>
          <p>{state.tag}</p>
        </div>
      )}
      <div className={classes.asideSection}>
        <span>Cloud provider</span>
        <p className={classes.capitalize}>
          {cloudProviderName(state.provider)}
        </p>
      </div>
      <div className={classes.asideSection}>
        <span>Cloud region</span>
        <p>
          {state.location?.native_code}: {state.location?.label}
        </p>
      </div>
      <div className={classes.asideSection}>
        <span>Instance type</span>
        <p className={classes.flexWrap}>
          {state.instanceType ? (
            <>
              {cluster && (
                <span
                  style={{
                    width: '100%',
                  }}
                >
                  Instances count: {state.numberOfInstances}
                </span>
              )}
              <span>{state.instanceType.native_name}: </span>
              <span>🔳 {state.instanceType.native_vcpus} CPU</span>
              <span>🧠 {state.instanceType.native_ram_gib} GiB RAM</span>
              <span>
                Price: {state.instanceType.native_reference_price_currency}
                {cluster
                  ? (
                      state.numberOfInstances *
                      state.instanceType.native_reference_price_hourly
                    )?.toFixed(4)
                  : state.instanceType.native_reference_price_hourly?.toFixed(
                      4,
                    )}{' '}
                hourly (~{state.instanceType.native_reference_price_currency}
                {cluster
                  ? (
                      state.numberOfInstances *
                      state.instanceType.native_reference_price_hourly *
                      MONTHLY_HOURS
                    ).toFixed(2)
                  : (
                      state.instanceType.native_reference_price_hourly *
                      MONTHLY_HOURS
                    ).toFixed(2)}{' '}
                per month)<sup>*</sup>
              </span>
            </>
          ) : (
            <span>No instance type available for this region.</span>
          )}
        </p>
      </div>
      <div className={classes.asideSection}>
        <span>Database volume</span>
        <p className={classes.flexWrap}>
          <span>Type: {state.volumeType}</span>
          <span>Size: {Number(state.storage)?.toFixed(2)} GiB </span>
        </p>
        <p>
          Price: {state.volumeCurrency}
          {cluster
            ? (state.volumePrice * state.numberOfInstances).toFixed(4)
            : state.volumePrice.toFixed(4)}{' '}
          hourly (~{state.volumeCurrency}
          {cluster
            ? (
                state.volumePrice *
                state.numberOfInstances *
                MONTHLY_HOURS
              ).toFixed(2)
            : (state.volumePrice * MONTHLY_HOURS).toFixed(2)}{' '}
          per month)
          <sup>*</sup>
        </p>
        <span className={classes.remark}>
          <sup>*</sup>Payment is made directly to the cloud provider. The
          estimated cost is calculated for the "
          {state.instanceType?.native_reference_price_region}" region and is not
          guaranteed. Please refer to{' '}
          <a
            href={pricingPageForProvider(state.provider)}
            target="_blank"
            rel="noreferrer"
          >
            the official pricing page
          </a>
          &nbsp; to confirm the actual costs. .
        </span>
      </div>
      <div className={classes.asideSection}>
        <span>
          Software: {cluster ? 'Postgres cluster' : 'DBLab SE'} (pay as you go)
        </span>
        <p className={classes.flexWrap}>
          {state.instanceType && (
            <>
              <span>Size: {state.instanceType.api_name}</span>
              <span>
                Price: $
                {cluster
                  ? (
                      state.instanceType.dle_se_price_hourly *
                      state.numberOfInstances
                    ).toFixed(4)
                  : state.instanceType.dle_se_price_hourly?.toFixed(4)}{' '}
                hourly (~$
                {cluster
                  ? (
                      state.numberOfInstances *
                      state.instanceType.dle_se_price_hourly *
                      MONTHLY_HOURS
                    ).toFixed(2)
                  : (
                      state.instanceType.dle_se_price_hourly * MONTHLY_HOURS
                    ).toFixed(2)}{' '}
                per month)
              </span>
            </>
          )}
        </p>
      </div>
      <Button
        variant="contained"
        color="primary"
        onClick={handleCreate}
        disabled={
          !state.name || (!cluster && !state.verificationToken) || disabled
        }
      >
        {cluster ? 'Create Postgres Cluster' : 'Create DBLab'}
      </Button>
    </div>
  )
}
