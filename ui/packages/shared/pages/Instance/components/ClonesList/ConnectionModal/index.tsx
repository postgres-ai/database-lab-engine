/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { observer } from 'mobx-react-lite'
import { makeStyles, IconButton } from '@material-ui/core'
import copy from 'copy-to-clipboard'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { Modal } from '@postgres.ai/shared/components/Modal'
import { TextField } from '@postgres.ai/shared/components/TextField'
import { icons } from '@postgres.ai/shared/styles/icons'
import {
  getSshPortForwardingCommand,
  getPsqlConnectionStr,
  getJdbcConnectionStr,
} from '@postgres.ai/shared/utils/connection'

type Props = {
  isOpen: boolean
  onClose: () => void
  cloneId: string
}

const useStyles = makeStyles({
  root: {
    fontSize: '14px',
  },
  item: {
    '& + $item': {
      marginTop: '16px',
    },
  },
  fieldText: {
    margin: 0,
  },
  field: {
    width: 'calc(100% - 24px)',
    margin: '16px 0 0 0',
  },
  fieldWrapper: {
    display: 'flex',
    alignItems: 'flex-end',
  },
  fieldInfo: {
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
    fontSize: '12px',
  },
})

export const ConnectionModal = observer((props: Props) => {
  const { isOpen, onClose, cloneId } = props

  const classes = useStyles()

  const stores = useStores()

  const { instance } = stores.main
  if (!instance) return null

  const clone = instance.state.cloning.clones.find(
    (clone) => clone.id === cloneId,
  )
  if (!clone) return null

  const sshPortForwardingCommand = getSshPortForwardingCommand(instance, clone)
  const psqlConnectionStr = getPsqlConnectionStr(clone)
  const jdbcConnectionStr = getJdbcConnectionStr(clone)

  return (
    <Modal title="Clone connection info" isOpen={isOpen} onClose={onClose}>
      <div className={classes.root}>
        {sshPortForwardingCommand && (
          <div className={classes.item}>
            <p className={classes.fieldText}>
              In a separate console, set up SSH port forwarding
              <br />
              (and keep it running):
            </p>
            <TextField
              label="SSH port forwarding"
              value={sshPortForwardingCommand}
              className={classes.field}
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
          </div>
        )}

        {psqlConnectionStr && (
          <div className={classes.item}>
            <p className={classes.fieldText}>Connect to Postgres using psql:</p>
            <div className={classes.fieldWrapper}>
              <TextField
                label="psql connection string"
                value={psqlConnectionStr}
                className={classes.field}
                InputProps={{
                  endAdornment: (
                    <IconButton
                      className={classes.copyButton}
                      onClick={() => copy(psqlConnectionStr)}
                    >
                      {icons.copyIcon}
                    </IconButton>
                  ),
                }}
              />

              <Tooltip
                content={
                  <>
                    Used to connect to Postgres using psql. Change DBNAME
                    to&nbsp; name of the database you want to connect. Use
                    PGPASSWORD&nbsp; environment variable to set database
                    password or type&nbsp; it when prompted.
                  </>
                }
              >
                <span className={classes.fieldInfo}>{icons.infoIcon}</span>
              </Tooltip>
            </div>
          </div>
        )}

        {jdbcConnectionStr && (
          <div className={classes.item}>
            <p className={classes.fieldText}>Connect to Postgres using JDBC:</p>
            <div className={classes.fieldWrapper}>
              <TextField
                label="JDBC connection string"
                value={jdbcConnectionStr}
                className={classes.field}
                InputProps={{
                  endAdornment: (
                    <IconButton
                      className={classes.copyButton}
                      onClick={() => copy(jdbcConnectionStr)}
                    >
                      {icons.copyIcon}
                    </IconButton>
                  ),
                }}
              />

              <Tooltip
                content={
                  <>
                    Used to connect to Postgres using JDBC. Change DBNAME
                    to&nbsp; name of the database you want to connect, change
                    DBPASSWORD&nbsp; to the password you’ve used on clone
                    creation.
                  </>
                }
              >
                <span className={classes.fieldInfo}>{icons.infoIcon}</span>
              </Tooltip>
            </div>
          </div>
        )}

        <p className={classes.note}>
          Password was set during clone creation. It’s not being stored.
          <br />
          You would need to recreate a clone if the password is lost.
        </p>
      </div>
    </Modal>
  )
})
