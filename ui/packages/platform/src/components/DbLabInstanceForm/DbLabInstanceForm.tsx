/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import {
  Checkbox,
  Grid,
  Button,
  TextField,
  FormControlLabel,
} from '@material-ui/core'
import CheckCircleOutlineIcon from '@material-ui/icons/CheckCircleOutline'
import BlockIcon from '@material-ui/icons/Block'
import WarningIcon from '@material-ui/icons/Warning'

import { styles } from '@postgres.ai/shared/styles/styles'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import {
  ClassesType,
  ProjectProps,
} from '@postgres.ai/platform/src/components/types'

import Actions from '../../actions/actions'
import ConsolePageTitle from './../ConsolePageTitle'
import Store from '../../stores/store'
import Urls from '../../utils/urls'
import { generateToken, isHttps } from '../../utils/utils'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { DbLabInstanceFormProps } from 'components/DbLabInstanceForm/DbLabInstanceFormWrapper'

interface DbLabInstanceFormWithStylesProps extends DbLabInstanceFormProps {
  classes: ClassesType
}

interface DbLabInstanceFormState {
  data: {
    auth: {
      token: string | null
    } | null
    projects: ProjectProps
    newDbLabInstance: {
      isUpdating: boolean
      isChecking: boolean
      isChecked: boolean
      isCheckProcessed: boolean
      errorMessage: string
      error: boolean
      isProcessed: boolean
      data: {
        id: string
      }
    } | null
    dbLabInstances: {
      isProcessing: boolean
      error: boolean
      isProcessed: boolean
      data: unknown
    } | null
  } | null
  url: string
  token: string | null
  useTunnel: boolean
  project: string
  errorFields: string[]
  sshServerUrl: string
}

class DbLabInstanceForm extends Component<
  DbLabInstanceFormWithStylesProps,
  DbLabInstanceFormState
> {
  state = {
    url: 'https://',
    token: null,
    useTunnel: false,
    project: this.props.project ? this.props.project : '',
    errorFields: [''],
    sshServerUrl: '',
    data: {
      auth: {
        token: null,
      },
      projects: {
        data: [],
        error: false,
        isProcessing: false,
        isProcessed: false,
      },
      newDbLabInstance: {
        isUpdating: false,
        isChecked: false,
        isChecking: false,
        isCheckProcessed: false,
        isProcessed: false,
        error: false,
        errorMessage: '',
        data: {
          id: '',
        },
      },
      dbLabInstances: {
        isProcessing: false,
        error: false,
        isProcessed: false,
        data: '',
      },
    },
  }

  unsubscribe: () => void
  componentDidMount() {
    const that = this
    const { orgId } = this.props

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data })
      const auth = this.data && this.data.auth ? this.data.auth : null
      const projects =
        this.data && this.data.projects ? this.data.projects : null
      const dbLabInstances =
        this.data && this.data.dbLabInstances ? this.data.dbLabInstances : null

      if (
        auth &&
        auth.token &&
        !projects.isProcessing &&
        !projects.error &&
        !projects.isProcessed
      ) {
        Actions.getProjects(auth.token, orgId)
      }

      if (
        auth &&
        auth.token &&
        !dbLabInstances?.isProcessing &&
        !dbLabInstances?.error &&
        !dbLabInstances?.isProcessed
      ) {
        Actions.getDbLabInstances(auth.token, orgId, 0)
      }
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
    Actions.resetNewDbLabInstance()
  }

  buttonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const data = this.state.data ? this.state.data.newDbLabInstance : null
    const errorFields = []

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
      this.state.project
    ) {
      Actions.addDbLabInstance(auth.token, {
        orgId: orgId,
        project: this.state.project,
        url: this.state.url,
        instanceToken: this.state.token,
        useTunnel: this.state.useTunnel,
        sshServerUrl: this.state.sshServerUrl,
      })
    }
  }

  checkUrlHandler = () => {
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const data = this.state.data ? this.state.data.newDbLabInstance : null
    const errorFields = []

    if (!this.state.url) {
      errorFields.push('url')
      return
    }

    if (auth && data && !data.isChecking && this.state.url) {
      Actions.checkDbLabInstanceUrl(
        auth.token,
        this.state.url,
        this.state.token,
        this.state.useTunnel,
      )
    }
  }

  returnHandler = () => {
    this.props.history.push(Urls.linkDbLabInstances(this.props))
  }

  processedHandler = () => {
    const data = this.state.data ? this.state.data.newDbLabInstance : null

    this.props.history.push(
      Urls.linkDbLabInstance(this.props, data?.data?.id as string),
    )
  }

  generateTokenHandler = () => {
    this.setState({ token: generateToken() })
  }

  render() {
    const { classes, orgPermissions } = this.props
    const data =
      this.state && this.state.data ? this.state.data.newDbLabInstance : null
    const projects =
      this.state && this.state.data && this.state.data.projects
        ? this.state.data.projects
        : null
    const projectsList = []
    const dbLabInstances =
      this.state && this.state.data && this.state.data.dbLabInstances
        ? this.state.data.dbLabInstances
        : null

    if (data && data.isProcessed && !data.error) {
      this.processedHandler()
      Actions.resetNewDbLabInstance()
    }

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        {...this.props}
        breadcrumbs={[
          { name: 'Database Lab Instances', url: 'instances' },
          { name: 'Add instance' },
        ]}
      />
    )

    const pageTitle = <ConsolePageTitle title="Add instance" />

    const permitted = !orgPermissions || orgPermissions.dblabInstanceCreate

    const instancesLoaded = dbLabInstances && dbLabInstances.data

    if (!projects || !projects.data || !instancesLoaded) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <PageSpinner />
        </div>
      )
    }

    if (projects.data && projects.data?.length > 0) {
      projects.data.map((p: { name: string; id: number }) => {
        return projectsList.push({ title: p.name, value: p.id })
      })
    }

    const isDataUpdating = data && (data.isUpdating || data.isChecking)

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {pageTitle}

        {!permitted && (
          <WarningWrapper>
            You do not have permission to add Database Lab instances.
          </WarningWrapper>
        )}

        <span>
          Database Lab provisioning is currently semi-automated.
          <br />
          First, you need to prepare a Database Lab instance on a separate&nbsp;
          machine. Once the instance is ready, register it here.
        </span>

        <div className={classes.errorMessage}>
          {data?.errorMessage ? data.errorMessage : null}
        </div>

        <Grid container>
          <div className={classes.fieldBlock}>
            <TextField
              disabled={!permitted}
              variant="outlined"
              id="project"
              label="Project"
              value={this.state.project}
              required
              className={classes.textField}
              onChange={(e) => {
                this.setState({
                  project: e.target.value,
                })
                Actions.resetNewDbLabInstance()
              }}
              margin="normal"
              error={this.state.errorFields.indexOf('project') !== -1}
              fullWidth
              inputProps={{
                name: 'project',
                id: 'project',
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

          <div className={classes.fieldBlock}>
            <TextField
              disabled={!permitted}
              variant="outlined"
              id="token"
              label="Verification token"
              value={this.state.token}
              required
              className={classes.textField}
              onChange={(e) => {
                this.setState({
                  token: e.target.value,
                })
                Actions.resetNewDbLabInstance()
              }}
              margin="normal"
              error={this.state.errorFields.indexOf('token') !== -1}
              fullWidth
              inputProps={{
                name: 'token',
                id: 'token',
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
              disabled={!permitted}
              variant="outlined"
              id="url"
              label="URL"
              required
              value={this.state.url}
              className={classes.textField}
              onChange={(e) => {
                this.setState({
                  url: e.target.value,
                })
                Actions.resetNewDbLabInstance()
              }}
              margin="normal"
              helperText={
                !isHttps(this.state.url) && !this.state.useTunnel ? (
                  <span>
                    <WarningIcon className={classes.warningIcon} />
                    <span className={classes.warning}>
                      The connection to the Database Lab API is not secure. Use
                      HTTPS.
                    </span>
                  </span>
                ) : null
              }
              error={this.state.errorFields.indexOf('url') !== -1}
              fullWidth
              inputProps={{
                name: 'url',
                id: 'url',
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
                    Actions.resetNewDbLabInstance()
                  }}
                  id="useTunnel"
                  name="useTunnel"
                />
              }
              label="Use tunnel"
              labelPlacement="end"
            />
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
                  Actions.resetNewDbLabInstance()
                }}
                margin="normal"
                error={this.state.errorFields.indexOf('sshServerUrl') !== -1}
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

            {data?.isCheckProcessed &&
            data?.isChecked &&
            (isHttps(this.state.url) || this.state.useTunnel) ? (
              <span className={classes.urlOk}>
                <CheckCircleOutlineIcon className={classes.urlOkIcon} />{' '}
                Verified
              </span>
            ) : null}

            {data?.isCheckProcessed &&
            data?.isChecked &&
            !isHttps(this.state.url) &&
            !this.state.useTunnel ? (
              <span className={classes.urlFail}>
                <BlockIcon className={classes.urlFailIcon} /> Verified but is
                not secure
              </span>
            ) : null}

            {data?.isCheckProcessed && !data?.isChecked ? (
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
              {this.props.edit ? 'Update' : 'Add'}
              {isDataUpdating && (
                <Spinner size="sm" className={classes.spinner} />
              )}
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

export default DbLabInstanceForm
