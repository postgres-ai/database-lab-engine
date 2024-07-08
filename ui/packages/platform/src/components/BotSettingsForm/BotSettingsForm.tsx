/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react'
import {
  Grid,
  Button,
  TextField,
  InputLabel,
  FormControl,
  Select,
  MenuItem,
  RadioGroup,
  FormControlLabel,
  Radio
} from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'
import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import ConsolePageTitle from '../ConsolePageTitle'
import { BotSettingsFormProps } from './BotSettingsFormWrapper'
import { GatewayLink } from "@postgres.ai/shared/components/GatewayLink";

interface BotSettingsFormWithStylesProps extends BotSettingsFormProps {
  classes: ClassesType
}

interface BotSettingState {
  custom_prompt: string
  model: string
  threadVisibility: string
  data: {
    auth: {
      token: string | null
    } | null
    orgProfile: {
      isUpdating: boolean
      error: boolean
      updateError: boolean
      errorMessage: string | undefined
      errorCode: number | undefined
      updateErrorMessage: string | null
      isProcessing: boolean
      orgId: number | null
      updateErrorFields: string[]
      data: {
        custom_prompt: string
        ai_model: string
      }
    } | null
  } | null
}



class BotSettingsForm extends Component<BotSettingsFormWithStylesProps, BotSettingState> {
  state = {
    custom_prompt: '',
    model: 'gemini-1.5-pro',
    threadVisibility: 'public',
    data: {
      auth: {
        token: null,
      },
      orgProfile: {
        isUpdating: false,
        isProcessing: false,
        error: false,
        updateError: false,
        errorMessage: undefined,
        errorCode: undefined,
        updateErrorMessage: null,
        updateErrorFields: [''],
        orgId: null,
        data: {
          custom_prompt: '',
          ai_model: 'gemini-1.5-pro',
        }
      }
    }
  }

  unsubscribe: Function
  componentDidMount() {
    const { orgId, mode } = this.props
    const that = this

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
       const auth = this.data && this.data.auth ? this.data.auth : null
       const orgProfile =
         this.data && this.data.orgProfile ? this.data.orgProfile : null

       that.setState({ data: this.data })
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  buttonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const data = this.state.data ? this.state.data.orgProfile : null

    if (auth) {
      let params: { custom_prompt?: string, model?: string } = {
        custom_prompt: this.state.custom_prompt,
        model: this.state.model
      };
      /*if (data.data.custom_prompt !== this.state.custom_prompt) {
        params.custom_prompt = this.state.custom_prompt;
      }*/
      if (data?.data?.ai_model !== this.state.model) {
        params.model = this.state.model;
      }
      Actions.updateAiBotSettings(auth.token, orgId, params)
    }
  }

  handleChangeModel = (event: React.ChangeEvent<{ value: unknown }>) => {
    this.setState({model: event.target.value as string})
  }

  handleChangeThreadVisibility = (event: React.ChangeEvent<{ value: unknown }>) => {
    this.setState({threadVisibility: event.target.value as string})
  }

  render() {
    const { classes, orgPermissions, mode } = this.props
    const orgId = this.props.orgId ? this.props.orgId : null


    return (
      <>
        <ConsoleBreadcrumbsWrapper
          org={this.props.org}
          project={this.props.project}
          breadcrumbs={[{ name: 'AI Bot' }]}
        />

        <ConsolePageTitle title="AI Bot settings" />

        {/*<div className={classes.errorMessage}>
          {data && data.updateErrorMessage ? data.updateErrorMessage : null}
        </div>*/}

        <Grid container spacing={3}>
          <Grid item xs={12} sm={12} lg={12} className={classes.container}>
            <Grid
              xs={12}
              sm={12}
              lg={8}
              item
              container
              direction={'column'}
            >
              <Grid item xs={12} sm={6}>
                <TextField
                  id="instructionsText"
                  label="Custom prompt"
                  fullWidth
                  minRows={4}
                  multiline
                  variant="outlined"
                  value={this.state.custom_prompt}
                  className={classes.instructionsField}
                  onChange={(e) => {
                    this.setState({
                      custom_prompt: e.target.value,
                    })
                  }}
                  /*error={
                    data?.updateErrorFields &&
                    data.updateErrorFields.indexOf('instructionsText') !== -1
                  }*/
                  margin="normal"
                  inputProps={{
                    name: 'instructionsText',
                    id: 'instructionsText',
                    shrink: 'true',
                  }}
                  InputLabelProps={{
                    shrink: true,
                    style: styles.inputFieldLabel,
                  }}
                  FormHelperTextProps={{
                    style: styles.inputFieldHelper,
                  }}
                  helperText={
                    <span>
                        Example: Our Postgres clusters are on AWS RDS, version is 15.
                    </span>
                  }
                />
                <FormControl fullWidth className={classes.selectField}>
                  <InputLabel id="model-radio-buttons-group-label">Model</InputLabel>
                  <RadioGroup
                    aria-labelledby="model-radio-buttons-group-label"
                    defaultValue="gemini-1.5-pro"
                    name="model-radio-buttons-group"
                    value={this.state.model}
                    onChange={this.handleChangeModel}
                  >
                    <FormControlLabel value="gemini-1.5-pro" control={<Radio />} label="gemini-1.5-pro" />
                    <FormControlLabel value="gpt-4-turbo" control={<Radio />} label="gpt-4-turbo	" />
                    <FormControlLabel disabled value="llama-3" control={<Radio />} label="Llama 3" />
                  </RadioGroup>
                </FormControl>
                <FormControl fullWidth className={classes.selectField}>
                  <InputLabel id="visibility-radio-buttons-group-label">Default thread visibility</InputLabel>
                  <RadioGroup
                    aria-labelledby="visibility-radio-buttons-group-label"
                    defaultValue="public"
                    name="visibility-radio-buttons-group"
                    value={this.state.threadVisibility}
                    onChange={this.handleChangeThreadVisibility}
                  >
                    <FormControlLabel value="public" control={<Radio />} label="Public" />
                    <FormControlLabel disabled value="private" control={<Radio />} label="Private" />
                  </RadioGroup>
                </FormControl>
              </Grid>
            </Grid>
            <Grid
              item
              xs={12}
              sm={12}
              lg={8}
              className={classes.updateButtonContainer}
            >
              <Button
                variant="contained"
                color="primary"
                //disabled={data?.isUpdating}
                id="orgSaveButton"
                onClick={this.buttonHandler}
              >
                Save
              </Button>
            </Grid>
          </Grid>
        </Grid>

        <div className={classes.bottomSpace} />
      </>
    )
  }
}

export default BotSettingsForm
