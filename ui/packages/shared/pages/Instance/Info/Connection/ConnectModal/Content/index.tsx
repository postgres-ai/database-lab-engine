/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles, IconButton } from '@material-ui/core'
import { observer } from 'mobx-react-lite'
import copy from 'copy-to-clipboard'

import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'
import { icons } from '@postgres.ai/shared/styles/icons'
import { Link } from '@postgres.ai/shared/components/Link2'
import { TextField } from '@postgres.ai/shared/components/TextField'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'

import { getCliInitCommand, getSshPortForwardingCommand } from './utils'

const useStyles = makeStyles({
  root: {
    fontSize: '14px',
  },
  list: {
    listStyle: 'decimal inside none',
    margin: 0,
    padding: 0,
  },
  item: {
    '& + $item': {
      marginTop: '16px',
    },
  },
  textField: {
    width: 'calc(100% - 24px)',
    margin: '16px 0 0 0',
  },
  textFieldWrapper: {
    display: 'flex',
    alignItems: 'flex-end',
  },
  textFieldInfo: {
    display: 'flex',
    alignItems: 'center',
    height: '32px',
    marginLeft: '12px',
  },
  copyButton: {
    width: '32px',
    height: '32px',
    padding: '8px',
  },
  note: {
    margin: '24px 0 0 0',
  },
})

export const Content = observer(() => {
  const classes = useStyles()
  const stores = useStores()

  const { instance } = stores.main
  if (!instance) return null

  const cliInitCommand = getCliInitCommand(instance)
  const sshPortForwardingCommand = getSshPortForwardingCommand(instance)
  const dblabStatusCommand = 'dblab instance status'

  return (
    <div className={classes.root}>
      <ol className={classes.list}>
        <li className={classes.item}>
          Generate a personal token on&nbsp;
          <Link to={
            // ROUTES.ORG.TOKENS.createPath({ org: context.org })
            '/'
            }>
            Access token page
          </Link>
          .
        </li>

        <li className={classes.item}>
          Use personal token to initialize a connection to the Database Lab
          instance:
          <div className={classes.textFieldWrapper}>
            <TextField
              label="CLI init command"
              value={cliInitCommand}
              className={classes.textField}
              InputProps={{
                endAdornment: (
                  <IconButton
                    className={classes.copyButton}
                    onClick={() => copy(cliInitCommand)}
                  >
                    {icons.copyIcon}
                  </IconButton>
                ),
              }}
            />
            <Tooltip
              content={
                <>
                  Use this oneliner to initialize Database Lab CLI on your
                  machine.
                  <br />
                  All UPPERCASED variables (if any) are to be substituted
                  <br />
                  by real values.
                </>
              }
            >
              <span className={classes.textFieldInfo}>
                {icons.infoIcon}
              </span>
            </Tooltip>
          </div>
        </li>

        {sshPortForwardingCommand && (
          <li className={classes.item}>
            In a separate console, set up SSH port forwarding (and keep it
            running):
            <TextField
              label="SSH port forwarding"
              value={sshPortForwardingCommand}
              className={classes.textField}
              InputProps={{
                endAdornment: (
                  <IconButton
                    className={classes.copyButton}
                    onClick={() => copy(sshPortForwardingCommand)}
                  >
                    {icons.copyIcon}
                  </IconButton>
                ),
              }}
            />
          </li>
        )}

        <li className={classes.item}>
          Test it:
          <TextField
            label="CLI status command"
            value={dblabStatusCommand}
            className={classes.textField}
            InputProps={{
              endAdornment: (
                <IconButton
                  className={classes.copyButton}
                  onClick={() => copy(dblabStatusCommand)}
                >
                  {icons.copyIcon}
                </IconButton>
              ),
            }}
          />
        </li>
      </ol>

      <p className={classes.note}>
        Read&nbsp;
        <GatewayLink href="https://postgres.ai/docs/database-lab/cli-reference#getting-started">
          the docs
        </GatewayLink>
        &nbsp;to get started with CLI.
      </p>
    </div>
  )
})
