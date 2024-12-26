import React from "react";
import AuditSettingsForm from "./AuditSettingsForm";

export interface AuditSettingsFormProps {
  mode?: string | undefined
  project?: string | undefined
  org?: string | number
  orgId?: number
  orgPermissions?: {
    settingsOrganizationUpdate?: boolean
  }
  orgData?: {
    priveleged_until: Date
    chats_private_allowed: boolean
    data?: {
      plan?: string
    } | null
  }
  match: {
    params: {
      project?: string
      projectId?: string | number | undefined
      org?: string
    }
  }
}



export const AuditSettingsFormWrapper = (props: AuditSettingsFormProps) => {
  return <AuditSettingsForm {...props} />
}
