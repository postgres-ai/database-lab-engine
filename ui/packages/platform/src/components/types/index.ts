import { RouteComponentProps } from 'react-router'
export interface ClassesType {
  [classes: string]: string
}
export interface QueryParamsType {
  session: string | undefined
  command: string | undefined
  author: string | undefined
  fingerprint: string | undefined
  project: string | undefined
  search: string | undefined
  is_favorite: string | undefined
}

export type OrgPermissions = { [permission: string]: boolean }

export interface MatchParams {
  project: string | undefined
  projectId: string | undefined
  org: string | undefined
  mode: string | undefined
  type: string | undefined
  fileType: string | undefined
  reportId: string | undefined
}

export interface Orgs {
  [project: string]: {
    alias: string
    is_blocked: boolean
    is_priveleged: boolean
    new_subscription: boolean
    is_blocked_on_creation: boolean
    stripe_payment_method_primary: string
    stripe_subscription_id: number
    priveleged_until: Date
    role: { id: number; permissions: string[] }
    name: string
    id: number
    owner_user_id: number
    projects: {
      [project: string]: {
        alias: string
        name: string
        id: number
        org_id: string
      }
    }
  }
}

export interface ProjectWrapperProps {
  classes: ClassesType
  location: RouteComponentProps['location']
  match: {
    params: {
      org?: string
      project?: string
      projectId?: string
    }
  }
  raw?: boolean
  org: string | number
  orgId: number
  userIsOwner: boolean
  orgPermissions: OrgPermissions
  auth: {
    isProcessed: boolean
    userId: number
    token: string
  } | null
  orgData: {
    projects: {
      [project: string]: {
        id: number
      }
    }
  }
  env: {
    data: {
      orgs?: Orgs
    }
  }
  envData: {
    orgs?: Orgs
  }
}

export interface OrganizationWrapperProps {
  classes: ClassesType
  match: { params: { org?: string, projectId?: number, project?: string } }
  location: RouteComponentProps['location']
  env: {
    data: {
      orgs?: Orgs
      info: {
        first_name: string
        user_name: string
        email: string
        is_tos_confirmed: boolean
        is_active: boolean
        id: number | null
      }
    }
  }
  auth: {
    isProcessed: boolean
    userId: number
    token: string
  } | null
  raw?: boolean
}

export interface OrganizationMenuProps {
  classes: { [classes: string]: string }
  location: RouteComponentProps['location']
  match: {
    params: {
      org?: string
      project?: string
      projectId?: string
    }
  }
  env: {
    data: {
      orgs?: Orgs
      info: {
        first_name: string
        user_name: string
        email: string
        is_tos_confirmed: boolean
        is_active: boolean
        id: number | null
      }
    }
  }
}

export interface UserProfile {
  data: {
    info: {
      first_name: string
      user_name: string
      email: string
      is_tos_confirmed: boolean
      is_active: boolean
    }
  }
  isConfirmProcessing: boolean
  isConfirmProcessed: boolean
  isTosAgreementConfirmProcessing: boolean
}

export interface TabPanelProps {
  children: React.ReactNode
  value: number
  index: number
}

export interface ProjectProps {
  error: boolean
  isProcessing: boolean
  isProcessed: boolean
  data: {
    name: string
    id: number
    project_label_or_name: string
  }[]
}

export interface TokenRequestProps {
  isProcessing: boolean
  isProcessed: boolean
  data: {
    name: string
    is_personal: boolean
    expires_at: string
    token: string
  }
  errorMessage: string
  error: boolean | null
}

export interface FlameGraphPlanType {
  [plan: string]: string | string[]
}
