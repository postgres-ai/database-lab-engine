import { SeImages } from '@postgres.ai/shared/types/api/endpoints/getSeImages'

import { dockerImagesConfig, genericImagePrefix } from './dockerCatalog'

// Mapping from the engine probe's provider key (probe.Provider) to a
// dockerImageOptions.type value the form already understands. Keys must
// match the values defined in engine/internal/retrieval/probe/provider.go
// (ProviderRDS, ProviderAurora, etc.) — see Task 18 in the plan.
const providerKeyToImageType: Record<string, string> = {
  generic: 'Generic Postgres',
  rds: 'rds',
  aurora: 'aurora',
  cloudsql: 'google-cloud-sql',
  supabase: 'supabase',
  heroku: 'heroku',
  timescale: 'timescale-cloud',
}

export type ProviderImageMapping = {
  imageType: string
  defaultTag?: string
  fallback: boolean
}

// Reads the most recent tag for a given PG major from the shared catalog.
// SE images (rds, aurora, etc.) require the platform-only getSeImages call
// and have no entry here — they return defaultTag: undefined.
const genericDefaultTag = (pgMajorVersion: number): string | undefined => {
  if (!pgMajorVersion) return undefined
  const version = String(pgMajorVersion)
  const tags = dockerImagesConfig[version]
  if (!tags || tags.length === 0) return undefined
  return `${version}-${tags[0]}`
}

// genericImagePathForVersion returns a pullable generic extended-postgres
// reference for a PG major version, or '' when the version is unknown. Simple
// mode uses it as the CE fallback for managed providers, whose SE images are
// only resolvable through the platform-only getSeImages call.
export const genericImagePathForVersion = (pgMajorVersion: number): string => {
  const tag = genericDefaultTag(pgMajorVersion)
  return tag ? `${genericImagePrefix}:${tag}` : ''
}

// ResolvedImage is the concrete image Simple mode ships for a probed source:
// either a platform SE image (carrying its curated preload-library preset) or
// the generic CE fallback. Apply and the preview both consume it, so they show
// and post the same thing.
export type ResolvedImage = {
  dockerPath: string
  dockerTag: string
  sharedPreloadLibraries?: string
  isSe: boolean
}

// selectSeImage picks the SE catalog entry matching the detected major version.
// Returns undefined when the catalog is empty (CE) or has no matching version,
// which signals the caller to fall back to the generic image.
export const selectSeImage = (
  seImages: SeImages[] | null | undefined,
  pgMajorVersion: number,
): SeImages | undefined =>
  seImages?.find((image) => image.pg_major_version === String(pgMajorVersion))

// resolveDockerImagePath returns the concrete image reference Simple mode ships
// for a probed source. Managed providers fall back to the generic image in CE,
// so both the Apply payload and the preview show the same value.
export const resolveDockerImagePath = (
  providerKey: string,
  pgMajorVersion: number,
  dockerTag?: string,
): string => {
  const mapping = providerKeyToImage(providerKey, pgMajorVersion)
  const tag = dockerTag || mapping.defaultTag || ''

  if (mapping.imageType === 'Generic Postgres') {
    return tag ? `${genericImagePrefix}:${tag}` : ''
  }

  return genericImagePathForVersion(pgMajorVersion)
}

// Resolves a probe provider key to a concrete docker image type the
// Configuration form can write into the projection. Unknown keys (including
// "azure", which has no matching SE image today) fall back to the generic
// Postgres image and set fallback=true so the UI can warn.
export const providerKeyToImage = (
  providerKey: string,
  pgMajorVersion: number,
): ProviderImageMapping => {
  const known = providerKeyToImageType[providerKey]

  if (!known) {
    return {
      imageType: 'Generic Postgres',
      defaultTag: genericDefaultTag(pgMajorVersion),
      fallback: true,
    }
  }

  if (known === 'Generic Postgres') {
    return {
      imageType: 'Generic Postgres',
      defaultTag: genericDefaultTag(pgMajorVersion),
      fallback: false,
    }
  }

  return { imageType: known, defaultTag: undefined, fallback: false }
}

export const dockerImageOptions = [
  {
    name: 'Generic PostgreSQL (postgresai/extended-postgres)',
    type: 'Generic Postgres',
  },
  { name: 'Generic PostgreSQL with PostGIS', type: 'postgis' },
  { name: 'Amazon RDS for PostgreSQL', type: 'rds' },
  { name: 'Amazon RDS Aurora for PostgreSQL', type: 'aurora' },
  { name: 'Heroku PostgreSQL', type: 'heroku' },
  { name: 'Supabase PostgreSQL', type: 'supabase' },
  { name: 'Google Cloud SQL for PostgreSQL', type: 'google-cloud-sql' },
  {
    name: 'Timescale Cloud',
    type: 'timescale-cloud',
  },
  { name: 'Custom image', type: 'custom' },
]

export const imagePgOptions = [
  {
    optionType: 'Generic Postgres',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'postgis',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'rds',
    pgDumpOptions: ['--exclude-schema=awsdms'],
    pgRestoreOptions: [],
  },
  {
    optionType: 'aurora',
    pgDumpOptions: ['--exclude-schema=awsdms'],
    pgRestoreOptions: [],
  },
  {
    optionType: 'heroku',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'supabase',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'google-cloud-sql',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
  {
    optionType: 'timescale-cloud',
    pgDumpOptions: [],
    pgRestoreOptions: [],
  },
]
