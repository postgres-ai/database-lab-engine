import {
  InstanceStateDto,
  formatInstanceStateDto,
} from '@postgres.ai/shared/types/api/entities/instanceState'

type CoreInstanceDto = {
  id: number
  state: InstanceStateDto
  plan: string
  selfassigned_instance_id: string
}

type CeInstanceDto = CoreInstanceDto & {}

type PlatformInstanceDto = CoreInstanceDto & {
  created_at: string
  telemetry_last_reported_at?: string
  created_formatted: string
  iid: number | null
  is_active: true
  org_id: number
  project_alias: string
  project_id: number
  project_name: string
  ssh_server_url: string | null
  url: string
  use_tunnel: boolean
  verify_token: string
}

export type InstanceDto = CeInstanceDto | PlatformInstanceDto

export const formatInstanceDto = (dto: InstanceDto) => {
  const coreMapped = {
    id: dto.id.toString(),
    state: formatInstanceStateDto(dto.state),
    dto,
  }

  const platformMapped =
    'created_at' in dto
      ? {
          createdAt: new Date(dto.created_at),
          ...(dto.telemetry_last_reported_at && {
            telemetryLastReportedAt: new Date(dto.telemetry_last_reported_at),
          }),
          createdFormatted: `${dto.created_formatted} UTC`,
          iid: dto.iid,
          isActive: dto.is_active,
          orgId: dto.org_id.toString(),
          projectAlias: dto.project_alias,
          projectId: dto.project_id.toString(),
          projectName: dto.project_name,
          sshServerUrl: dto.ssh_server_url,
          useTunnel: dto.use_tunnel,
          verifyToken: dto.verify_token,
          url: dto.url,
        }
      : null

  return {
    ...coreMapped,
    ...platformMapped,
  }
}

export type Instance = ReturnType<typeof formatInstanceDto>
