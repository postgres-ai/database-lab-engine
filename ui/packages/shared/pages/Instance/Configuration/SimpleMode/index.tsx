import { useState } from 'react'
import { observer } from 'mobx-react-lite'
import { Button, TextField, Typography } from '@material-ui/core'
import Box from '@mui/material/Box'

import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { SeImages } from '@postgres.ai/shared/types/api/endpoints/getSeImages'
import { ProposedConfig } from '@postgres.ai/shared/types/api/endpoints/probeSource'

import {
  ResolvedImage,
  providerKeyToImage,
  resolveDockerImagePath,
  selectSeImage,
} from '../configOptions'
import { FormValues } from '../useForm'
import { PreviewCard } from './PreviewCard'

type Props = {
  instanceId: string
  disabled?: boolean
  onApplied?: () => void
  onEdit?: (proposed: ProposedConfig, password: string) => void
}

// genericFallbackImage resolves the generic CE image for a probed source. Used
// when no SE image is available (CE, or a managed provider with no SE image for
// the detected major version).
const genericFallbackImage = (proposed: ProposedConfig): ResolvedImage => {
  const mapping = providerKeyToImage(
    proposed.dockerImage,
    proposed.pgMajorVersion,
  )
  const isGeneric = mapping.imageType === 'Generic Postgres'

  return {
    dockerPath: resolveDockerImagePath(
      proposed.dockerImage,
      proposed.pgMajorVersion,
      proposed.dockerTag,
    ),
    dockerTag: isGeneric ? proposed.dockerTag || mapping.defaultTag || '' : '',
    sharedPreloadLibraries: proposed.sharedPreloadLibraries,
    isSe: false,
  }
}

// Translates a ProposedConfig from POST /admin/probe-source into the
// FormValues shape the Expert form uses. Both Apply (→ updateConfig) and
// Edit (→ formik.setValues) consume it, so the engine receives the same
// projection regardless of which flow the user picks. The resolved image (SE
// or generic) is passed in so the preview and the applied config never diverge.
export const buildProjectionFromProposed = (
  proposed: ProposedConfig,
  password: string,
  resolved?: ResolvedImage,
): FormValues => {
  const mapping = providerKeyToImage(
    proposed.dockerImage,
    proposed.pgMajorVersion,
  )
  const isGeneric = mapping.imageType === 'Generic Postgres'
  const image = resolved ?? genericFallbackImage(proposed)

  return {
    debug: false,
    dockerImage: isGeneric
      ? String(proposed.pgMajorVersion)
      : mapping.imageType,
    dockerImageType: mapping.imageType,
    dockerPath: image.dockerPath,
    dockerTag: image.dockerTag,
    sharedBuffers: proposed.sharedBuffers,
    sharedPreloadLibraries:
      image.sharedPreloadLibraries ?? proposed.sharedPreloadLibraries,
    // tuningParams is typed as string on FormValues but updateConfig.ts
    // spreads it as a key-value object; cast matches the Expert form's
    // formatTuningParamsToObj(...) as unknown as string pattern.
    tuningParams: { ...proposed.queryTuning } as unknown as string,
    timetable: '0 0 * * *',
    dbname: proposed.source.dbname,
    host: proposed.source.host,
    port: String(proposed.source.port),
    username: proposed.source.username,
    password,
    databases: proposed.databases.join(' '),
    dumpParallelJobs: '',
    dumpIgnoreErrors: false,
    restoreParallelJobs: '',
    // managed sources reference cloud-only extensions/objects (e.g. supabase_vault)
    // that are absent from the clone image; best-effort restore skips them.
    restoreIgnoreErrors: true,
    restoreConfigs: '',
    pgDumpCustomOptions: '',
    // skip ownership and privileges on restore: managed sources (Supabase, RDS)
    // reference roles that do not exist in the clone, which would abort pg_restore.
    pgRestoreCustomOptions: '--no-owner --no-privileges --no-tablespaces',
    retrievalMode: 'logical',
    physicalTool: '',
    physicalDockerImage: '',
    physicalSyncEnabled: false,
    physicalWalgBackupName: '',
    physicalPgbackrestStanza: '',
    physicalPgbackrestDelta: false,
    physicalEnvs: [],
  }
}

// resolveProbeImage selects the platform SE image for a managed provider when
// the SE catalog is reachable — SE/Enterprise instances expose it (platformUrl
// set), CE returns undefined. Falls back to the generic image for CE and for
// managed providers with no SE image at the detected major version.
export const resolveProbeImage = async (
  proposed: ProposedConfig,
  getSeImages: (args: {
    packageGroup: string
  }) => Promise<SeImages[] | null | undefined>,
): Promise<ResolvedImage> => {
  const mapping = providerKeyToImage(
    proposed.dockerImage,
    proposed.pgMajorVersion,
  )

  if (mapping.imageType !== 'Generic Postgres') {
    const seImages = await getSeImages({ packageGroup: mapping.imageType })
    const se = selectSeImage(seImages, proposed.pgMajorVersion)

    if (se) {
      return {
        dockerPath: se.location,
        dockerTag: se.tag,
        sharedPreloadLibraries: se.pg_config_presets?.shared_preload_libraries,
        isSe: true,
      }
    }
  }

  return genericFallbackImage(proposed)
}

export const SimpleMode = observer(
  ({ instanceId, disabled, onApplied, onEdit }: Props) => {
    const stores = useStores()
    const main = stores.main

    const [url, setUrl] = useState('')
    const [password, setPassword] = useState('')
    const [probing, setProbing] = useState(false)
    const [probeError, setProbeError] = useState<string | null>(null)
    const [proposed, setProposed] = useState<ProposedConfig | null>(null)
    const [resolved, setResolved] = useState<ResolvedImage | null>(null)
    const [applying, setApplying] = useState(false)
    const [applyError, setApplyError] = useState<string | null>(null)

    const onDetect = async () => {
      setProbing(true)
      setProbeError(null)
      setProposed(null)
      setResolved(null)
      setApplyError(null)

      const result = await main.probeSource({ url, password })

      if (!result) {
        setProbing(false)
        setProbeError('Probe is not available on this instance.')
        return
      }
      if (result.error) {
        setProbing(false)
        setProbeError(result.error.message)
        return
      }
      if (result.response) {
        const image = await resolveProbeImage(result.response, main.getSeImages)
        setResolved(image)
        setProposed(result.response)
      }

      setProbing(false)
    }

    const onApply = async () => {
      if (!proposed || !resolved) return

      setApplying(true)
      setApplyError(null)

      try {
        const projection = buildProjectionFromProposed(proposed, password, resolved)
        const response = await main.updateConfig(projection, instanceId)

        if (!response) {
          setApplyError(
            main.configError ?? 'Could not apply the proposed configuration.',
          )
          return
        }

        const refresh = await main.fullRefresh(instanceId)

        if (refresh?.error) {
          setApplyError(
            `Configuration applied, but starting data retrieval failed: ${refresh.error.message}`,
          )
          return
        }

        onApplied?.()
      } catch (err) {
        setApplyError(
          err instanceof Error
            ? err.message
            : 'Could not apply the proposed configuration.',
        )
      } finally {
        setApplying(false)
      }
    }

    const handleEdit = () => {
      if (proposed) onEdit?.(proposed, password)
    }

    const canDetect =
      !probing && url.trim().length > 0 && password.length > 0 && !disabled

    return (
      <Box mt={2} mb={2} data-testid="simple-mode">
        {!proposed && (
          <Box>
            <Typography variant="h6">Simple configuration</Typography>
            <Typography variant="body2">
              Paste your source connection string and password. We'll probe
              the source, propose a configuration, and let you review before
              starting retrieval.
            </Typography>

            <Box mt={2}>
              <TextField
                label="Connection string"
                placeholder="postgres://user@host:5432/dbname"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                fullWidth
                disabled={probing || disabled}
                inputProps={{ 'data-testid': 'simple-url' }}
              />
            </Box>

            <Box mt={2}>
              <TextField
                label="Password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                fullWidth
                disabled={probing || disabled}
                inputProps={{ 'data-testid': 'simple-password' }}
              />
            </Box>

            <Box mt={2}>
              <Button
                variant="contained"
                color="secondary"
                onClick={onDetect}
                disabled={!canDetect}
                data-testid="simple-detect"
              >
                Detect &amp; preview
                {probing && <Spinner size="sm" />}
              </Button>
            </Box>

            {probeError && (
              <Box
                mt={1}
                color="#d32f2f"
                fontSize={13}
                data-testid="probe-error"
              >
                {probeError}
              </Box>
            )}
          </Box>
        )}

        {proposed && resolved && (
          <PreviewCard
            proposed={proposed}
            resolved={resolved}
            applying={applying}
            applyError={applyError}
            onApply={onApply}
            onEdit={handleEdit}
          />
        )}
      </Box>
    )
  },
)
