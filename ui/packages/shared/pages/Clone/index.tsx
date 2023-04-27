/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useEffect, useState } from 'react'
import { observer } from 'mobx-react-lite'
import { useHistory } from 'react-router-dom'
import copyToClipboard from 'copy-to-clipboard'
import {
  makeStyles,
  Button,
  FormControlLabel,
  Checkbox,
  TextField,
  IconButton,
} from '@material-ui/core'

import {
  getSshPortForwardingCommand,
  getPsqlConnectionStr,
  getJdbcConnectionStr,
} from '@postgres.ai/shared/utils/connection'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { DestroyCloneRestrictionModal } from '@postgres.ai/shared/components/DestroyCloneRestrictionModal'
import { DestroyCloneModal } from '@postgres.ai/shared/components/DestroyCloneModal'
import { ResetCloneModal } from '@postgres.ai/shared/components/ResetCloneModal'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { round } from '@postgres.ai/shared/utils/numbers'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { icons } from '@postgres.ai/shared/styles/icons'
import { styles } from '@postgres.ai/shared/styles/styles'
import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'

import { Status } from './Status'
import { useCreatedStores } from './useCreatedStores'
import { Host } from './context'
import {
  getCliDestroyCloneCommand,
  getCliProtectedCloneCommand,
  getCliResetCloneCommand,
  getCreateSnapshotCommand,
} from './utils'

const textFieldWidth = 525

const useStyles = makeStyles(
  (theme) => ({
    wrapper: {
      display: 'flex',
      gap: '60px',
      maxWidth: '1200px',
      fontSize: '14px !important',
      marginTop: '20px',

      '@media (max-width: 1300px)': {
        flexDirection: 'column',
        gap: '20px',
      },
    },
    title: {
      marginTop: '16px',
    },
    tooltip: {
      marginTop: '8px',
    },
    container: {
      maxWidth: textFieldWidth + 25,
      marginTop: '16px',
    },
    text: {
      marginTop: '4px',
    },
    errorStub: {
      marginTop: '24px',
    },
    spinner: {
      marginLeft: '8px',
    },
    summary: {
      flex: '1 1 0',
      minWidth: 0,
    },
    snippetContainer: {
      flex: '1 1 0',
      minWidth: 0,
      boxShadow: 'rgba(0, 0, 0, 0.1) 0px 4px 12px',
      padding: '10px 20px 10px 20px',
      height: 'max-content',
      borderRadius: '4px',
    },
    paramTitle: {
      display: 'inline-block',
      width: 200,
    },
    readyStatus: {
      color: 'green',
    },
    failedStatus: {
      color: 'red',
    },
    startingStatus: {
      color: 'orange',
    },
    fieldBlock: {
      width: '100%',
      maxWidth: textFieldWidth + 25,
      position: 'relative',
    },
    saveButton: {
      padding: 8,
      marginLeft: 10,
      marginBottom: -56,
    },
    actions: {
      display: 'flex',
      flexWrap: 'wrap',
      rowGap: '16px',
      marginBottom: '20px',
    },
    actionButton: {
      marginRight: '16px',
    },
    remark: {
      fontSize: '12px',
      lineHeight: '14px',
      maxWidth: textFieldWidth,
      display: 'inline-block',
    },
    textField: {
      ...styles.inputField,
      'max-width': textFieldWidth,
      display: 'inline-block',
      '& .MuiOutlinedInput-input': {
        paddingRight: '32px!important',
      },
    },
    errorMessage: {
      color: 'red',
      marginBottom: 10,
    },
    textFieldInfo: {
      position: 'absolute',
      top: '25px',
      marginLeft: '5px',
      display: 'inline-block',
      [theme.breakpoints.down('sm')]: {
        display: 'none',
      },
    },
    checkBoxInfo: {
      'margin-top': '0px',
      'margin-left': '3px',
      display: 'inline-block',
      '& svg': {
        verticalAlign: 'middle',
      },
    },
    checkboxLabel: {
      marginRight: '0px',
    },
    copyFieldContainer: {
      position: 'relative',
      display: 'inline-block',
      maxWidth: textFieldWidth,
      width: '100%',
    },
    status: {
      maxWidth: `${textFieldWidth}px`,
    },
    copyButton: {
      position: 'absolute',
      top: 16,
      right: 0,
      zIndex: 100,
      width: 32,
      height: 32,
      padding: 8,
    },
  }),
  { index: 1 },
)

type Props = Host

export const Clone = observer((props: Props) => {
  const classes = useStyles()
  const history = useHistory()
  const stores = useCreatedStores(props)

  // Clone modals.
  const [isOpenRestrictionModal, setIsOpenRestrictionModal] = useState(false)
  const [isOpenDestroyModal, setIsOpenDestroyModal] = useState(false)
  const [isOpenResetModal, setIsOpenResetModal] = useState(false)

  // Initial loading data.
  useEffect(() => {
    stores.main.load(props.instanceId, props.cloneId)
  }, [])

  const {
    instance,
    snapshots,
    clone,
    isResettingClone,
    isDestroyingClone,
    isReloading,
    isUpdatingClone,
    isCloneStable,
  } = stores.main

  const headRendered = (
    <>
      {/* //TODO(Anton): make global reset styles. */}
      <style>{'p { margin: 0;}'}</style>

      {props.elements.breadcrumbs}

      <SectionTitle
        className={classes.title}
        tag="h1"
        level={1}
        text={`Clone ${props.cloneId}`}
      />
    </>
  )

  // Getting instance error.
  if (stores.main.instanceError)
    return (
      <>
        {headRendered}

        <ErrorStub
          title={stores.main.instanceError.title}
          message={stores.main.instanceError.message}
        />
      </>
    )

  // Getting clone error.
  if (stores.main.cloneError)
    return (
      <>
        {headRendered}

        <ErrorStub {...stores.main.cloneError} />
      </>
    )

  // Initial getting data spinner.
  if (!instance || !clone) {
    return (
      <>
        {headRendered}

        <PageSpinner />
      </>
    )
  }

  // Clone reset.
  const requestResetClone = () => setIsOpenResetModal(true)

  const resetClone = (snapshotId: string) => stores.main.resetClone(snapshotId)

  // Clone destroy.
  const requestDestroyClone = () => {
    if (clone.protected) {
      setIsOpenRestrictionModal(true)
    } else {
      setIsOpenDestroyModal(true)
    }
  }

  const destroyClone = async () => {
    const isSuccess = await stores.main.destroyClone()
    if (isSuccess) history.push(props.routes.instance())
  }

  // Clone reload.
  const reloadClone = () => stores.main.reload()

  // Data protection.
  const toggleDataProtection = () => stores.main.updateClone(!clone.protected)

  // Commands.
  const sshPortForwardingUrl = getSshPortForwardingCommand(instance, clone)
  const jdbcConnectionStr = getJdbcConnectionStr(clone)
  const psqlConnectionStr = getPsqlConnectionStr(clone)

  const hasConnectionInfo =
    sshPortForwardingUrl || jdbcConnectionStr || psqlConnectionStr

  // Controls.
  const isDisabledControls =
    isResettingClone ||
    isDestroyingClone ||
    isReloading ||
    isUpdatingClone ||
    !isCloneStable

  return (
    <>
      {headRendered}
      <div className={classes.wrapper}>
        {stores.main.resetCloneError && (
          <ErrorStub
            title="Resetting error"
            message={stores.main.resetCloneError}
            className={classes.errorStub}
          />
        )}

        {stores.main.destroyCloneError && (
          <ErrorStub
            title="Destroying error"
            message={stores.main.destroyCloneError}
            className={classes.errorStub}
          />
        )}

        <div className={classes.summary}>
          <div className={classes.actions}>
            <Button
              variant="contained"
              color="primary"
              onClick={requestResetClone}
              disabled={isDisabledControls}
              title={'Reset clone'}
              className={classes.actionButton}
            >
              Reset clone
              {isResettingClone && (
                <Spinner size="sm" className={classes.spinner} />
              )}
            </Button>
            <Button
              variant="contained"
              color="primary"
              onClick={requestDestroyClone}
              disabled={isDisabledControls}
              title={'Destroy this clone'}
              className={classes.actionButton}
            >
              Destroy clone
              {isDestroyingClone && (
                <Spinner size="sm" className={classes.spinner} />
              )}
            </Button>
            <Button
              variant="outlined"
              color="secondary"
              onClick={reloadClone}
              disabled={isDisabledControls}
              title={'Refresh clone information'}
              className={classes.actionButton}
            >
              Reload info
              {isReloading && <Spinner size="sm" className={classes.spinner} />}
            </Button>
          </div>
          <div>
            <p>
              <strong>Created</strong>
            </p>
            <p className={classes.text}>{clone.createdAt}</p>
          </div>
          <br />
          <div>
            <p>
              <strong>Data state at</strong>&nbsp;
              <Tooltip
                content={
                  <>
                    <strong>Data state time</strong> is a time at which data
                    is&nbsp; recovered for this clone.
                  </>
                }
              >
                {icons.infoIcon}
              </Tooltip>
            </p>
            <p className={classes.text}>{clone.snapshot?.dataStateAt}</p>
          </div>
          <br />
          <div>
            <p>
              <strong>Status</strong>
            </p>

            <Status rawClone={clone} className={classes.status} />
          </div>
          <br />
          <div>
            <p>
              <strong>Summary</strong>&nbsp;
              <Tooltip
                content={
                  <>
                    <strong>Logical data size</strong> is a logical size of
                    files in PGDATA which is being thin-cloned.
                    <br />
                    <br />
                    <strong>Physical data diff size</strong> is an actual size
                    of a&nbsp; thin clone. On creation there is no diff between
                    clone’s&nbsp; and initial data state, all data blocks match.
                    During work&nbsp; with the clone data diverges and data diff
                    size increases.
                    <br />
                    <br />
                    <strong>Clone creation time</strong> is time which was&nbsp;
                    spent to provision the clone.
                  </>
                }
              >
                {icons.infoIcon}
              </Tooltip>
            </p>

            <p className={classes.text}>
              <span className={classes.paramTitle}>Logical data size:</span>
              {instance.state.dataSize
                ? formatBytesIEC(instance.state.dataSize)
                : '-'}
            </p>

            <p className={classes.text}>
              <span className={classes.paramTitle}>
                Physical data diff size:
              </span>
              {clone.metadata.cloneDiffSize
                ? formatBytesIEC(clone.metadata.cloneDiffSize)
                : '-'}
            </p>

            <p className={classes.text}>
              <span className={classes.paramTitle}>Clone creation time:</span>
              {clone.metadata.cloningTime
                ? `${round(clone.metadata.cloningTime, 2)} s`
                : '-'}
            </p>
          </div>

          <br />

          {hasConnectionInfo && (
            <>
              <p>
                <strong>Connection info</strong>
              </p>

              {sshPortForwardingUrl && (
                <div className={classes.fieldBlock}>
                  In a separate console, set up SSH port forwarding (and keep it
                  running):
                  <div className={classes.copyFieldContainer}>
                    <TextField
                      variant="outlined"
                      label="SSH port forwarding"
                      value={sshPortForwardingUrl}
                      className={classes.textField}
                      margin="normal"
                      fullWidth
                      // @ts-ignore
                      readOnly
                      InputLabelProps={{
                        shrink: true,
                        style: styles.inputFieldLabel,
                      }}
                      FormHelperTextProps={{
                        style: styles.inputFieldHelper,
                      }}
                    />
                    <IconButton
                      className={classes.copyButton}
                      aria-label="Copy"
                      onClick={() => copyToClipboard(sshPortForwardingUrl)}
                    >
                      {icons.copyIcon}
                    </IconButton>
                  </div>
                </div>
              )}

              {psqlConnectionStr && (
                <div className={classes.fieldBlock}>
                  <div className={classes.copyFieldContainer}>
                    <TextField
                      variant="outlined"
                      id="psqlConnStr"
                      label="psql connection string"
                      value={psqlConnectionStr}
                      className={classes.textField}
                      margin="normal"
                      fullWidth
                      // @ts-ignore
                      readOnly
                      InputLabelProps={{
                        shrink: true,
                        style: styles.inputFieldLabel,
                      }}
                      FormHelperTextProps={{
                        style: styles.inputFieldHelper,
                      }}
                    />
                    <IconButton
                      className={classes.copyButton}
                      aria-label="Copy"
                      onClick={() => copyToClipboard(psqlConnectionStr)}
                    >
                      {icons.copyIcon}
                    </IconButton>
                  </div>
                  &nbsp;
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
                    <span className={classes.textFieldInfo}>
                      {icons.infoIcon}
                    </span>
                  </Tooltip>
                </div>
              )}

              {jdbcConnectionStr && (
                <div className={classes.fieldBlock}>
                  <div className={classes.copyFieldContainer}>
                    <TextField
                      variant="outlined"
                      label="JDBC connection string"
                      value={jdbcConnectionStr}
                      className={classes.textField}
                      margin="normal"
                      fullWidth
                      // @ts-ignore
                      readOnly
                      InputLabelProps={{
                        shrink: true,
                        style: styles.inputFieldLabel,
                      }}
                      FormHelperTextProps={{
                        style: styles.inputFieldHelper,
                      }}
                    />
                    <IconButton
                      className={classes.copyButton}
                      aria-label="Copy"
                      onClick={() => copyToClipboard(jdbcConnectionStr)}
                    >
                      {icons.copyIcon}
                    </IconButton>
                  </div>
                  &nbsp;
                  <Tooltip
                    content={
                      <>
                        Used to connect to Postgres using JDBC. Change DBNAME
                        to&nbsp; name of the database you want to connect,
                        change DBPASSWORD&nbsp; to the password you’ve used on
                        clone creation.
                      </>
                    }
                  >
                    <span className={classes.textFieldInfo}>
                      {icons.infoIcon}
                    </span>
                  </Tooltip>
                </div>
              )}
            </>
          )}
          <br />
          <div className={classes.fieldBlock}>
            <span className={classes.remark}>
              Password was set during clone creation. It’s not being stored.
              <br />
              You would need to recreate a clone if the password is lost.
            </span>
          </div>
          <br />
          <p>
            <strong>Protection</strong>
          </p>
          <p>
            <FormControlLabel
              className={classes.checkboxLabel}
              control={
                <Checkbox
                  checked={clone.protected}
                  onChange={toggleDataProtection}
                  name="protected"
                  disabled={isDisabledControls}
                />
              }
              label="Enable deletion protection"
            />
            <br />
            <span className={classes.remark}>
              When enabled no one can delete this clone and automated deletion
              is also disabled.
              <br />
              Please be careful: abandoned clones with this checkbox enabled may
              cause out-of-disk-space events. Check disk space on daily basis
              and delete this clone once the work is done.
            </span>
          </p>
          {stores.main.updateCloneError && (
            <ErrorStub
              title="Updating error"
              message={stores.main.updateCloneError}
              className={classes.errorStub}
            />
          )}
        </div>

        <div className={classes.snippetContainer}>
          <SectionTitle tag="h2" level={2} text={'Reset clone using the CLI'} />
          <p className={classes.tooltip}>
            You can reset the clone using the CLI using the following command:
          </p>
          <SyntaxHighlight content={getCliResetCloneCommand(props.cloneId)} />

          <SectionTitle
            className={classes.title}
            tag="h2"
            level={2}
            text={'Destroy clone using the CLI'}
          />
          <p className={classes.tooltip}>
            You can destroy the clone using the CLI using the following command:
          </p>
          <SyntaxHighlight content={getCliDestroyCloneCommand(props.cloneId)} />

          <SectionTitle
            className={classes.title}
            tag="h2"
            level={2}
            text={'Toggle deletion protection using the CLI'}
          />
          <p className={classes.tooltip}>
            You can toggle deletion protection using the CLI for this clone
            using the following command:
          </p>
          <SyntaxHighlight content={getCliProtectedCloneCommand(true)} />

          <SyntaxHighlight content={getCliProtectedCloneCommand(false)} />

          <SectionTitle
            className={classes.title}
            tag="h2"
            level={2}
            text={'Create snapshot for this clone using the CLI'}
          />
          <p className={classes.tooltip}>
            You can create a snapshot for this clone using the CLI using the
            following command:
          </p>
          <SyntaxHighlight content={getCreateSnapshotCommand(props.cloneId)} />
        </div>

        <>
          <DestroyCloneRestrictionModal
            isOpen={isOpenRestrictionModal}
            onClose={() => setIsOpenRestrictionModal(false)}
            cloneId={clone.id}
          />

          <DestroyCloneModal
            isOpen={isOpenDestroyModal}
            onClose={() => setIsOpenDestroyModal(false)}
            cloneId={clone.id}
            onDestroyClone={destroyClone}
          />

          <ResetCloneModal
            isOpen={isOpenResetModal}
            onClose={() => setIsOpenResetModal(false)}
            clone={clone}
            snapshots={snapshots.data}
            onResetClone={resetClone}
            version={instance.state.engine.version}
          />
        </>
      </div>
    </>
  )
})
