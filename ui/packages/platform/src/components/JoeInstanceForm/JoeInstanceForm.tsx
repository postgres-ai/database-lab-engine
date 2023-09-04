/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import { Checkbox, Grid, Button, TextField } from '@material-ui/core'
import CheckCircleOutlineIcon from '@material-ui/icons/CheckCircleOutline'
import BlockIcon from '@material-ui/icons/Block'
import WarningIcon from '@material-ui/icons/Warning'

import { styles } from '@postgres.ai/shared/styles/styles'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import ConsolePageTitle from './../ConsolePageTitle'
import Urls from '../../utils/urls'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import FormControlLabel from '@material-ui/core/FormControlLabel'
import { generateToken, isHttps } from 'utils/utils'
import { JoeInstanceFormProps } from 'components/JoeInstanceForm/JoeInstanceFormWrapper'

interface JoeInstanceFormWithStylesProps extends JoeInstanceFormProps {
  classes: ClassesType
}

interface JoeInstanceFormState {
  data: {
    auth: {
      token: string
    } | null
    newJoeInstance: {
      isUpdating: boolean
      isChecking: boolean
      isCheckProcessed: boolean
      isChecked: boolean
      isProcessed: boolean
      isProcessing: boolean
      error: boolean
      errorMessage: string | null
    } | null
    joeInstances: {
      isProcessed: boolean
      isProcessing: boolean
      error: boolean
      data: unknown | null
    } | null
  } | null
  url: string
  useTunnel: boolean
  token: string
  project: string
  errorFields: string[]
  sshServerUrl: string
}

class JoeInstanceForm extends Component<
  JoeInstanceFormWithStylesProps,
  JoeInstanceFormState
> {
  state = {
    url: 'https://',
    useTunnel: false,
    token: '',
    project: this.props.project ? this.props.project : '',
    errorFields: [''],
    sshServerUrl: '',
    data: {
      auth: {
        token: '',
      },
      newJoeInstance: {
        isUpdating: false,
        isChecking: false,
        isCheckProcessed: false,
        isChecked: false,
        isProcessed: false,
        isProcessing: false,
        error: false,
        errorMessage: null,
      },
      joeInstances: {
        isProcessed: false,
        isProcessing: false,
        error: false,
        data: null,
      },
    },
  }

  unsubscribe: Function
  componentDidMount() {
    const that = this
    const { orgId } = this.props

    this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      that.setState({ data: this.data })
      const auth = this.data && this.data.auth ? this.data.auth : null
      const joeInstances =
        this.data && this.data.joeInstances ? this.data.joeInstances : null

      if (
        auth &&
        auth.token &&
        !joeInstances?.isProcessing &&
        !joeInstances?.error &&
        !joeInstances?.isProcessed
      ) {
        Actions.getJoeInstances(auth.token, orgId, 0)
      }
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
    Actions.resetNewJoeInstance()
  }

  buttonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const data = this.state.data ? this.state.data.newJoeInstance : null
    let errorFields: JoeInstanceFormState['errorFields'] = []

    if (!this.state.url) {
      errorFields.push('url')
    }

    if (!this.state.project) {
      errorFields.push('project')
    }

    if (!this.state.token) {
      errorFields.push('token')
    }

    if (errorFields.length > 0) {
      this.setState({ errorFields: errorFields })
      return
    }

    this.setState({ errorFields: [] })

    if (
      auth &&
      data &&
      !data.isUpdating &&
      this.state.url &&
      this.state.token &&
      this.state.token &&
      this.state.project
    ) {
      Actions.addJoeInstance(auth.token, {
        orgId: orgId,
        project: this.state.project,
        url: this.state.url,
        verifyToken: this.state.token,
        useTunnel: this.state.useTunnel,
        sshServerUrl: this.state.sshServerUrl,
      })
    }
  }

  checkUrlHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const data = this.state.data ? this.state.data.newJoeInstance : null
    let errorFields: JoeInstanceFormState['errorFields'] = []

    if (!this.state.url) {
      errorFields.push('url')
      return
    }

    if (auth && data && !data.isChecking && this.state.url) {
      Actions.checkJoeInstanceUrl(auth.token, {
        orgId: orgId,
        project: this.state.project,
        url: this.state.url,
        verifyToken: this.state.token,
        useTunnel: this.state.useTunnel,
      })
    }
  }

  returnHandler = () => {
    this.props.history.push(Urls.linkJoeInstances(this.props))
  }

  generateTokenHandler = () => {
    this.setState({ token: generateToken() })
  }

  render() {
    const { classes, orgPermissions } = this.props
    const data =
      this.state && this.state.data ? this.state.data.newJoeInstance : null

    if (data && data.isProcessed && !data.error) {
      this.returnHandler()
      Actions.resetNewJoeInstance()
    }

    const joeInstances =
      this.state && this.state.data && this.state.data.joeInstances
        ? this.state.data.joeInstances
        : null

    const permitted = !orgPermissions || orgPermissions.joeInstanceCreate

    const instancesLoaded = joeInstances !== null && joeInstances?.data

    if (!data || !instancesLoaded) {
      return (
        <div>
          <Spinner size="lg" className={classes.progress} />
        </div>
      )
    }

    const isDataUpdating = data && (data.isUpdating || data.isChecking)

    return (
      <div className={classes.root}>
        <ConsoleBreadcrumbsWrapper
          org={this.props.org}
          project={this.props.project}
          breadcrumbs={[
            { name: 'SQL Optimization', url: null },
            { name: 'Joe Instances', url: 'joe-instances' },
            { name: 'Add instance' },
          ]}
        />

        <ConsolePageTitle title="Add instance" />

        {!permitted && (
          <WarningWrapper>
            You do not have permission to add Joe instances.
          </WarningWrapper>
        )}

        <span>
          Joe provisioning is currently semi-automated.
          <br />
          First, you need to prepare a Joe instance on a separate&nbsp; machine.
          Once the instance is ready, register it here.
        </span>

        <div className={classes.errorMessage}>
          {data.errorMessage ? data.errorMessage : null}
        </div>

        <Grid container>
          <div className={classes.fieldBlock}>
            <TextField
              variant="outlined"
              id="project"
              disabled={!permitted}
              label="Project"
              value={this.state.project}
              required
              className={classes.textField}
              onChange={(e) => {
                this.setState({
                  project: e.target.value,
                })
                Actions.resetNewJoeInstance()
              }}
              margin="normal"
              error={this.state.errorFields.indexOf('project') !== -1}
              fullWidth
              inputProps={{
                name: 'project',
                id: 'project',
                shrink: 'true',
              }}
              InputLabelProps={{
                shrink: true,
                style: styles.inputFieldLabel,
              }}
              FormHelperTextProps={{
                style: styles.inputFieldHelper,
              }}
            />
          </div>

          <div className={classes.fieldBlock}>
            <TextField
              variant="outlined"
              id="token"
              disabled={!permitted}
              label="Signing secret"
              value={this.state.token}
              required
              className={classes.textField}
              onChange={(e) => {
                this.setState({
                  token: e.target.value,
                })
                Actions.resetNewJoeInstance()
              }}
              margin="normal"
              error={this.state.errorFields.indexOf('token') !== -1}
              fullWidth
              inputProps={{
                name: 'token',
                id: 'token',
                shrink: 'true',
              }}
              InputLabelProps={{
                shrink: true,
                style: styles.inputFieldLabel,
              }}
              FormHelperTextProps={{
                style: styles.inputFieldHelper,
              }}
            />
            <div>
              <Button
                variant="contained"
                color="primary"
                disabled={isDataUpdating || !permitted}
                onClick={this.generateTokenHandler}
              >
                Generate
              </Button>
            </div>
          </div>

          <div className={classes.fieldBlock} style={{ marginTop: 10 }}>
            <TextField
              variant="outlined"
              id="url"
              disabled={!permitted}
              label="URL"
              required
              value={this.state.url}
              className={classes.textField}
              onChange={(e) => {
                this.setState({
                  url: e.target.value,
                })
                Actions.resetNewJoeInstance()
              }}
              margin="normal"
              helperText={
                this.state.url &&
                !isHttps(this.state.url) &&
                !this.state.useTunnel ? (
                  <span>
                    <WarningIcon className={classes.warningIcon} />
                    <span className={classes.warning}>
                      The connection to the Joe API is not secure. Use HTTPS.
                    </span>
                  </span>
                ) : null
              }
              error={this.state.errorFields.indexOf('url') !== -1}
              fullWidth
              inputProps={{
                name: 'url',
                id: 'url',
                shrink: 'true',
              }}
              InputLabelProps={{
                shrink: true,
                style: styles.inputFieldLabel,
              }}
              FormHelperTextProps={{
                style: styles.inputFieldHelper,
              }}
            />

            <div className={classes.fieldBlock}>
              <FormControlLabel
                control={
                  <Checkbox
                    disabled={!permitted}
                    checked={this.state.useTunnel}
                    onChange={(e) => {
                      this.setState({
                        useTunnel: e.target.checked,
                      })
                      Actions.resetNewJoeInstance()
                    }}
                    id="useTunnel"
                    name="useTunnel"
                  />
                }
                label="Use tunnel"
                labelPlacement="end"
              />
              {this.state.useTunnel && (
                <div>
                  <TextField
                    variant="outlined"
                    id="token"
                    label="SSH server URL"
                    value={this.state.sshServerUrl}
                    className={classes.textField}
                    onChange={(e) => {
                      this.setState({
                        sshServerUrl: e.target.value,
                      })
                      Actions.resetNewJoeInstance()
                    }}
                    margin="normal"
                    error={
                      this.state.errorFields.indexOf('sshServerUrl') !== -1
                    }
                    fullWidth
                    inputProps={{
                      name: 'sshServerUrl',
                      id: 'sshServerUrl',
                      shrink: true,
                    }}
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>
              )}
            </div>
          </div>

          <div>
            <Button
              variant="contained"
              color="primary"
              disabled={isDataUpdating || !permitted}
              onClick={this.checkUrlHandler}
            >
              Verify URL
            </Button>

            {data.isCheckProcessed &&
            data.isChecked &&
            (isHttps(this.state.url) || this.state.useTunnel) ? (
              <span className={classes.urlOk}>
                <CheckCircleOutlineIcon className={classes.urlOkIcon} />{' '}
                Verified
              </span>
            ) : null}

            {data.isCheckProcessed &&
            data.isChecked &&
            !isHttps(this.state.url) &&
            !this.state.useTunnel ? (
              <span className={classes.urlFail}>
                <BlockIcon className={classes.urlFailIcon} /> Verified but is
                not secure
              </span>
            ) : null}

            {data.isCheckProcessed && !data.isChecked ? (
              <span className={classes.urlFail}>
                <BlockIcon className={classes.urlFailIcon} /> Not available
              </span>
            ) : null}
          </div>

          <div className={classes.fieldBlock} style={{ marginTop: 40 }}>
            <Button
              variant="contained"
              color="primary"
              disabled={isDataUpdating || !permitted}
              onClick={this.buttonHandler}
            >
              Add
            </Button>
            &nbsp;&nbsp;
            <Button
              variant="contained"
              color="primary"
              disabled={isDataUpdating || !permitted}
              onClick={this.returnHandler}
            >
              Cancel
            </Button>
          </div>
        </Grid>
      </div>
    )
  }
}

export default JoeInstanceForm
