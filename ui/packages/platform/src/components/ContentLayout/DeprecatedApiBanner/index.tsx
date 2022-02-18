import { makeStyles } from '@material-ui/core'

import { Status } from '@postgres.ai/shared/components/Status'
import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'
import { colors } from '@postgres.ai/shared/styles/vars'

const useStyles = makeStyles({
  root: {
    background: colors.status.warning,
    color: colors.white,
    fontSize: '12px',
    padding: '4px 10px',
    lineHeight: '1.5',
  },
  status: {
    color: 'inherit',
  },
  link: {
    color: 'inherit',
  },
})

export const DeprecatedApiBanner = () => {
  const classes = useStyles()

  return (
    <div className={classes.root}>
      <Status type="warning" className={classes.status} disableColor>
        The version of your DLE instance is deprecated.
      </Status>{' '}
      Some information about DLE, disks, clones, and snapshots may be
      unavailable.
      <br />
      Please upgrade your DLE to&nbsp;
      <GatewayLink
        href="https://gitlab.com/postgres-ai/database-lab/-/releases"
        className={classes.link}
      >
        the latest available version
      </GatewayLink>
      .
    </div>
  )
}
