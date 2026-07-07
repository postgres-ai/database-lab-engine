import { Button, Link, Typography } from '@material-ui/core'
import Box from '@mui/material/Box'

import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ProposedConfig } from '@postgres.ai/shared/types/api/endpoints/probeSource'

import { ResolvedImage, providerKeyToImage } from '../configOptions'

type Props = {
  proposed: ProposedConfig
  resolved: ResolvedImage
  applying: boolean
  applyError: string | null
  onApply: () => void
  onEdit: () => void
}

const Field = ({ label, value }: { label: string; value: string }) => (
  <Box display="flex" mb={0.5}>
    <Box minWidth={220} fontWeight={600}>
      {label}
    </Box>
    <Box minWidth={0} style={{ overflowWrap: 'anywhere' }}>
      {value}
    </Box>
  </Box>
)

const Callout = ({ children }: { children: React.ReactNode }) => (
  <Box
    mt={1}
    p={1}
    bgcolor="#fff8e1"
    borderLeft="4px solid #f5a623"
    fontSize={13}
    style={{ overflowWrap: 'anywhere' }}
  >
    {children}
  </Box>
)

export const PreviewCard = ({
  proposed,
  resolved,
  applying,
  applyError,
  onApply,
  onEdit,
}: Props) => {
  const mapping = providerKeyToImage(
    proposed.dockerImage,
    proposed.pgMajorVersion,
  )
  const preloadLibraries =
    resolved.sharedPreloadLibraries ?? proposed.sharedPreloadLibraries
  const tuningEntries = Object.entries(proposed.queryTuning ?? {})

  return (
    <Box
      mt={2}
      p={2}
      border="1px solid #e0e0e0"
      borderRadius={4}
      data-testid="preview-card"
    >
      <Typography variant="h6">Proposed configuration</Typography>

      <Box mt={2}>
        <Field
          label="Detected provider"
          value={proposed.detectedProvider || 'unknown'}
        />
        <Field
          label="Docker image"
          value={resolved.dockerPath || mapping.imageType}
        />
        <Field
          label="Postgres major version"
          value={String(proposed.pgMajorVersion || 'unknown')}
        />
        <Field
          label="Databases"
          value={proposed.databases?.join(', ') || '(none)'}
        />
        <Field label="shared_buffers" value={proposed.sharedBuffers || ''} />
        <Field
          label="shared_preload_libraries"
          value={preloadLibraries || ''}
        />
      </Box>

      {tuningEntries.length > 0 && (
        <Box mt={2}>
          <Typography variant="subtitle2">Query tuning</Typography>
          <Box component="table" mt={1} style={{ borderCollapse: 'collapse' }}>
            <tbody>
              {tuningEntries.map(([k, v]) => (
                <tr key={k}>
                  <td style={{ padding: '2px 16px 2px 0', fontWeight: 600 }}>
                    {k}
                  </td>
                  <td style={{ padding: '2px 0' }}>{v}</td>
                </tr>
              ))}
            </tbody>
          </Box>
        </Box>
      )}

      <Box mt={2}>
        {(mapping.fallback || proposed.detectedProvider === 'generic') && (
          <Callout>
            Could not detect a managed cloud provider; using the generic
            Postgres image. Switch to Expert mode if your source runs on a
            managed service and we missed it.
          </Callout>
        )}
        {!proposed.memoryProbed && (
          <Callout>
            Could not detect host memory; <code>shared_buffers</code> is set
            to a 1&nbsp;GB safe default. Adjust in Expert mode if your host
            has more RAM.
          </Callout>
        )}
        <Callout>
          Query tuning is copied from your source. If you use the RDS refresh
          tool, these values may not match production — review in Expert mode
          after the first retrieval run.
        </Callout>
        {resolved.isSe ? (
          <Callout>
            Shipping the <code>{resolved.dockerPath}</code> image with its
            curated <code>shared_preload_libraries</code> preset, so the
            detected extensions are bundled.
          </Callout>
        ) : (
          <Callout>
            We'll ship <code>{preloadLibraries}</code>. If the chosen image
            does not bundle one of these libraries, the clone container will
            fail to start with a "could not load library" error — check{' '}
            <code>docker logs dblab_server</code> after Apply.
          </Callout>
        )}
      </Box>

      <Box mt={2} display="flex" alignItems="center">
        <Button
          variant="contained"
          color="secondary"
          onClick={onApply}
          disabled={applying}
          data-testid="preview-apply"
        >
          Apply &amp; start retrieval
          {applying && <Spinner size="sm" />}
        </Button>
        <Box ml={2}>
          <Link
            component="button"
            type="button"
            onClick={onEdit}
            data-testid="preview-edit"
          >
            Edit before applying
          </Link>
        </Box>
      </Box>

      {applyError && (
        <Box mt={1} color="#d32f2f" fontSize={13} data-testid="apply-error">
          {applyError}
        </Box>
      )}
    </Box>
  )
}
