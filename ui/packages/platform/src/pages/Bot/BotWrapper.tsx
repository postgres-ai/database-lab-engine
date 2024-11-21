import { BotPage } from "./index";
import {RouteComponentProps} from "react-router";
import {AlertSnackbarProvider} from "@postgres.ai/shared/components/AlertSnackbar/useAlertSnackbar";
import { AiBotProvider } from "./hooks";

export interface BotWrapperProps {
  orgId?: number;
  envData: {
    info?: {
      user_name?: string
    }
  };
  orgData: {
    id: number,
    is_chat_public_by_default: boolean
    data: {
      plan: string
    } | null
  },
  history: RouteComponentProps['history']
  project?: string
  match: {
    params: {
      org?: string
      threadId?: string
      projectId?: string | number | undefined
    }
  }
}


export const BotWrapper = (props: BotWrapperProps) => {
  return (
    <AlertSnackbarProvider>
      <AiBotProvider
        args={{
          threadId: props.match.params.threadId,
          orgId: props.orgData.id,
          isPublicByDefault: props.orgData.is_chat_public_by_default
      }}>
        <BotPage {...props} />
      </AiBotProvider>
    </AlertSnackbarProvider>
  )
}
