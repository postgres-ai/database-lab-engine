import { simpleInstallRequest } from 'helpers/simpleInstallRequest'
import { initialState } from 'components/DbLabInstanceForm/reducer'
import { DEBUG_API_SERVER } from 'components/DbLabInstanceForm/utils'

const API_SERVER = process.env.REACT_APP_API_SERVER

const formatExtraEnvs = (extraEnvs: { [key: string]: string }) => {
  return Object.entries(extraEnvs)
    .filter(([key, value]) => value)
    .map(([key, value]) => {
      if (key === 'GCP_SERVICE_ACCOUNT_CONTENTS') {
        return `${key}=${value.replace(/\n\s+/g, '')}`
      }
      return `${key}=${value}`
    })
}

export const launchDeploy = async ({
  state,
  userID,
  orgKey,
  extraEnvs,
  cloudImage,
}: {
  state: typeof initialState
  orgKey: string
  userID?: number
  extraEnvs: {
    [key: string]: string
  }
  cloudImage: string
}) => {
  const response = await simpleInstallRequest(
    '/launch',
    {
      method: 'POST',
      body: JSON.stringify({
        playbook: 'deploy_dle.yml',
        provision: state.provider,
        server: {
          name: state.name,
          serverType: state.instanceType.native_name,
          image: cloudImage,
          location: state.location.native_code,
        },
        image: 'postgresai/dle-se-ansible:v1.0-rc.8',
        extraVars: [
          `provision=${state.provider}`,
          `server_name=${state.name}`,
          `dle_platform_project_name=${state.name}`,
          `server_type=${state.instanceType.native_name}`,
          `server_image=${cloudImage}`,
          `server_location=${state.location.native_code}`,
          `volume_size=${state.storage}`,
          `dle_version=${state.tag}`,
          `zpool_datasets_number=${state.snapshots}`,
          `dle_verification_token=${state.verificationToken}`,
          `dle_platform_org_key=${orgKey}`,
          ...(state.publicKeys
            ? // eslint-disable-next-line no-useless-escape
              [`ssh_public_keys=\"${state.publicKeys}\"`]
            : []),
          ...(API_SERVER === DEBUG_API_SERVER
            ? [`dle_platform_url=https://v2.postgres.ai/api/general`]
            : []),
        ],
        extraEnvs: formatExtraEnvs(extraEnvs),
      }),
    },
    userID,
  )

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : await response.json(),
  }
}
