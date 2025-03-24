import React from "react";
import DBLabSettingsForm from "./DBLabSettingsForm";

export interface DBLabSettingsFormProps {
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
    consulting_type: string | null
  }
  match: {
    params: {
      project?: string
      projectId?: string | number | undefined
      org?: string
    }
  }
}



export const DBLabSettingsFormWrapper = (props: DBLabSettingsFormProps) => {
  return <DBLabSettingsForm {...props} />
}
