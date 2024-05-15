/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/core'
import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'

import settings from 'utils/settings'

const useStyles = makeStyles(
  (theme) => ({
    footer: {
      flex: '0 0 auto',
      backgroundColor: 'rgb(68, 79, 96)',
      color: '#fff',
      display: 'flex',
      justifyContent: 'center',
      padding: '16px 20px',
      [theme.breakpoints.down('sm')]: {
        padding: '12px 12px',
        flexDirection: 'column',
      },
    },
    footerCopyrightItem: {
      marginRight: 50,
      [theme.breakpoints.down('sm')]: {
        marginBottom: 10,
      },
    },
    footerLinks: {
      display: 'flex',
      [theme.breakpoints.down('sm')]: {
        flexDirection: 'column',
        flexWrap: 'wrap',
        maxHeight: '80px',
      },
    },
    footerItem: {
      marginLeft: 10,
      marginRight: 10,
      color: '#fff',
      '& a': {
        color: '#fff',
        textDecoration: 'none',
      },
      '& a:hover': {
        textDecoration: 'none',
      },
      [theme.breakpoints.down('sm')]: {
        marginLeft: 0,
        marginBottom: 5,
      },
    },
    footerItemSeparator: {
      display: 'inline-block',
      [theme.breakpoints.down('sm')]: {
        display: 'none',
      },
    },
  }),
  { index: 1 },
)

export const Footer = () => {
  const classes = useStyles()

  return (
    <div className={classes.footer}>
      <div className={classes.footerCopyrightItem}>
        {new Date().getFullYear()} © Postgres.AI
      </div>
      <div className={classes.footerLinks}>
        <div className={classes.footerItem}>
          <GatewayLink href={settings.rootUrl + '/docs'}>
            Documentation
          </GatewayLink>
        </div>
        <div className={classes.footerItemSeparator}>|</div>
        <div className={classes.footerItem}>
          <GatewayLink href={settings.rootUrl + '/blog'}>
            News
          </GatewayLink>
        </div>
        <div className={classes.footerItemSeparator}>|</div>
        <div className={classes.footerItem}>
          <GatewayLink href={settings.rootUrl + '/tos'}>
            Terms of Service
          </GatewayLink>
        </div>
        <div className={classes.footerItemSeparator}>|</div>
        <div className={classes.footerItem}>
          <GatewayLink href={settings.rootUrl + '/privacy'}>
            Privacy Policy
          </GatewayLink>
        </div>
        <div className={classes.footerItemSeparator}>|</div>
        <div className={classes.footerItem}>
          <GatewayLink href={settings.rootUrl + '/contact'}>
            Ask support
          </GatewayLink>
        </div>
      </div>
    </div>
  )
}
