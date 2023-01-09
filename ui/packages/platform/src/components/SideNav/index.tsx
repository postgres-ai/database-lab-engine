/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { NavLink } from 'react-router-dom'
import { ListItem, List } from '@material-ui/core'
import { makeStyles } from '@material-ui/styles'

import { colors } from '@postgres.ai/shared/styles/colors'

import { ROUTES } from 'config/routes'

const useStyles = makeStyles(
  {
    listItem: {
      padding: 0,
    },
    navLink: {
      textDecoration: 'none',
      color: '#000000',
      fontWeight: 'bold',
      fontSize: '14px',
      width: '100%',
      padding: '12px 14px',
    },
    activeNavLink: {
      backgroundColor: colors.consoleStroke,
    },
  },
  { index: 1 },
)

export const SideNav = () => {
  const classes = useStyles()

  return (
    <List component="nav">
      <ListItem button className={classes.listItem}>
        <NavLink
          className={classes.navLink}
          activeClassName={classes.activeNavLink}
          to={ROUTES.ROOT.path}
          exact
        >
          Organizations
        </NavLink>
      </ListItem>
      <ListItem button className={classes.listItem}>
        <NavLink
          className={classes.navLink}
          activeClassName={classes.activeNavLink}
          to={ROUTES.PROFILE.path}
        >
          Profile
        </NavLink>
      </ListItem>
    </List>
  )
}
