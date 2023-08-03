export type GetSeImages = (args: {
  packageGroup: string
  platformUrl?: string
}) => Promise<{
  response: InstanceRetrievalType | null
  error: Response | null
}>

export interface SeImages {
  org_id?: number
  package_group: string
  pg_major_version: string
  tag: string
  pg_config_presets?: {
    shared_preload_libraries: string
  }
  location: string
}

export const formatSeImages = (seImages: SeImages[]) => seImages

export type InstanceRetrievalType = ReturnType<typeof formatSeImages>
