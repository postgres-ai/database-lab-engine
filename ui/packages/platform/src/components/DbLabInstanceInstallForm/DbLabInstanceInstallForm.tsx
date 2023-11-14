/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useReducer } from 'react'
import { TextField, Button } from '@material-ui/core'

import ConsolePageTitle from '../ConsolePageTitle'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { ClassesType } from '@postgres.ai/platform/src/components/types'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { DbLabInstanceFormProps } from 'components/DbLabInstanceForm/DbLabInstanceFormWrapper'
import { initialState, reducer } from 'components/DbLabInstanceForm/reducer'
import { DbLabInstanceFormInstallSidebar } from 'components/DbLabInstanceInstallForm/DbLabInstanceInstallFormSidebar'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinnerFlex'
import { AnsibleInstance } from 'components/DbLabInstanceInstallForm/DbLabFormSteps/AnsibleInstance'
import { DockerInstance } from 'components/DbLabInstanceInstallForm/DbLabFormSteps/DockerInstance'
import { availableTags } from 'components/DbLabInstanceForm/utils'
import { Select } from '@postgres.ai/shared/components/Select'

import { generateToken, validateDLEName } from 'utils/utils'
import urls from 'utils/urls'

interface DbLabInstanceFormWithStylesProps extends DbLabInstanceFormProps {
  classes: ClassesType
}

const DbLabInstanceInstallForm = (props: DbLabInstanceFormWithStylesProps) => {
  const { classes, orgPermissions } = props
  const [state, dispatch] = useReducer(reducer, initialState)

  const permitted = !orgPermissions || orgPermissions.dblabInstanceCreate

  const pageTitle = <ConsolePageTitle title="Install DBLab" />
  const breadcrumbs = (
    <ConsoleBreadcrumbsWrapper
      {...props}
      breadcrumbs={[
        { name: 'Database Lab Instances', url: 'instances' },
        { name: 'Install DBLab' },
      ]}
    />
  )

  const handleGenerateToken = () => {
    dispatch({
      type: 'change_verification_token',
      verificationToken: generateToken(),
    })
  }

  const handleReturnToList = () => {
    props.history.push(urls.linkDbLabInstances(props))
  }

  const handleSetFormStep = (step: string) => {
    dispatch({ type: 'set_form_step', formStep: step })
  }

  const handleReturnToForm = () => {
    dispatch({ type: 'set_form_step', formStep: initialState.formStep })
  }

  if (state.isLoading) return <StubSpinner />

  return (
    <div className={classes.root}>
      {breadcrumbs}

      {pageTitle}

      {!permitted && (
        <WarningWrapper>
          You do not have permission to add Database Lab instances.
        </WarningWrapper>
      )}

      <div className={classes.container}>
        {state.formStep === initialState.formStep && permitted ? (
          <>
            <div className={classes.form}>
              <p className={classes.sectionTitle}>1. Provide DBLab name</p>
              <TextField
                required
                label="DBLab Name"
                variant="outlined"
                fullWidth
                value={state.name}
                className={classes.marginTop}
                InputLabelProps={{
                  shrink: true,
                }}
                helperText={
                  validateDLEName(state.name)
                    ? 'Name must be lowercase and contain only letters and numbers.'
                    : ''
                }
                error={validateDLEName(state.name)}
                onChange={(
                  event: React.ChangeEvent<
                    HTMLTextAreaElement | HTMLInputElement
                  >,
                ) =>
                  dispatch({ type: 'change_name', name: event.target.value })
                }
              />
              <p className={classes.sectionTitle}>
                2. Define DBLab verification token (keep it secret!)
              </p>
              <div className={classes.generateContainer}>
                <TextField
                  required
                  label="DBLab Verification Token"
                  variant="outlined"
                  fullWidth
                  value={state.verificationToken}
                  className={classes.marginTop}
                  InputLabelProps={{
                    shrink: true,
                  }}
                  onChange={(
                    event: React.ChangeEvent<
                      HTMLTextAreaElement | HTMLInputElement
                    >,
                  ) =>
                    dispatch({
                      type: 'change_verification_token',
                      verificationToken: event.target.value,
                    })
                  }
                />
                <Button
                  variant="contained"
                  color="primary"
                  disabled={!permitted}
                  onClick={handleGenerateToken}
                >
                  Generate random
                </Button>
              </div>
              <p className={classes.sectionTitle}>
                3. Choose DBLab server version
              </p>
              <Select
                label="Select tag"
                items={
                  availableTags.map((tag) => {
                    const defaultTag = availableTags[0]

                    return {
                      value: tag,
                      children: defaultTag === tag ? `${tag} (default)` : tag,
                    }
                  }) ?? []
                }
                value={state.tag}
                onChange={(
                  e: React.ChangeEvent<HTMLTextAreaElement | HTMLInputElement>,
                ) =>
                  dispatch({
                    type: 'set_tag',
                    tag: e.target.value,
                  })
                }
              />
              <p className={classes.sectionTitle}>
                4. Provide SSH public keys (one per line)
              </p>{' '}
              <p className={classes.instanceParagraph}>
                The specified ssh public keys will be added to authorized_keys
                on the DBLab server. Add your public key here to have access to
                the server after deployment.
              </p>
              <TextField
                label="SSH public keys"
                variant="outlined"
                fullWidth
                multiline
                value={state.publicKeys}
                helperText={
                  state.publicKeys && state.publicKeys.length < 30
                    ? 'Public key is too short'
                    : ''
                }
                error={state.publicKeys && state.publicKeys.length < 30}
                className={classes.marginTop}
                InputLabelProps={{
                  shrink: true,
                }}
                onChange={(
                  event: React.ChangeEvent<
                    HTMLTextAreaElement | HTMLInputElement
                  >,
                ) =>
                  dispatch({
                    type: 'change_public_keys',
                    publicKeys: event.target.value,
                  })
                }
              />
            </div>
            <DbLabInstanceFormInstallSidebar
              state={state}
              disabled={
                validateDLEName(state.name) ||
                (state.publicKeys && state.publicKeys.length < 30)
              }
              handleCreate={() =>
                !validateDLEName(state.name) && handleSetFormStep('docker')
              }
            />
          </>
        ) : state.formStep === 'ansible' && permitted ? (
          <AnsibleInstance
            state={state}
            orgId={props.orgId}
            formStep={state.formStep}
            setFormStep={handleSetFormStep}
            goBack={handleReturnToList}
            goBackToForm={handleReturnToForm}
          />
        ) : state.formStep === 'docker' && permitted ? (
          <DockerInstance
            state={state}
            orgId={props.orgId}
            formStep={state.formStep}
            setFormStep={handleSetFormStep}
            goBack={handleReturnToList}
            goBackToForm={handleReturnToForm}
          />
        ) : null}
      </div>
    </div>
  )
}

export default DbLabInstanceInstallForm
