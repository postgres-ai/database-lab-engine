import React from "react";
import BotSettingsForm from "./BotSettingsForm";

export interface BotSettingsFormProps {
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



export const BotSettingsFormWrapper = (props: BotSettingsFormProps) => {
  return <BotSettingsForm {...props} />
}
