/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/core'

import settings from 'utils/settings'
import Link from '@material-ui/core/Link'

const useStyles = makeStyles((theme) => ({
  footer: {
    flex: '0 0 auto',
    backgroundColor: 'rgb(68, 79, 96)',
    color: '#fff',
    display: 'flex',
    justifyContent: 'center',
    padding: '16px 20px',
    [theme.breakpoints.down('sm')]: {
      padding: '16px 12px',
      flexDirection: 'column'
    }
  },
  footerCopyrightItem: {
    marginRight: 50,
    [theme.breakpoints.down('sm')]: {
      marginBottom: 10,
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
}))

export const Footer = () => {
  const classes = useStyles()

  return (
    <div className={classes.footer}>
      <div className={classes.footerCopyrightItem}>{new Date().getFullYear()} © Postgres.ai</div>
      <div className={classes.footerItem}>
        <Link href={settings.rootUrl + '/docs'} target="_blank">
          Documentation
        </Link>
      </div>
      <div className={classes.footerItemSeparator}>|</div>
      <div className={classes.footerItem}>
        <Link href={settings.rootUrl + '/blog'} target="_blank">
          News
        </Link>
      </div>
      <div className={classes.footerItemSeparator}>|</div>
      <div className={classes.footerItem}>
        <Link href={settings.rootUrl + '/tos'} target="_blank">
          Terms of Service
        </Link>
      </div>
      <div className={classes.footerItemSeparator}>|</div>
      <div className={classes.footerItem}>
        <Link href={settings.rootUrl + '/privacy'} target="_blank">
          Privacy Policy
        </Link>
      </div>
      <div className={classes.footerItemSeparator}>|</div>
      <div className={classes.footerItem}>
        <Link href={settings.rootUrl + '/contact'} target="_blank">
          Ask support
        </Link>
      </div>
    </div>
  )
}
