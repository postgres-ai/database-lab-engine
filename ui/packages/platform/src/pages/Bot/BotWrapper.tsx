import { BotPage } from "./index";
import {RouteComponentProps} from "react-router";
import {AlertSnackbarProvider} from "@postgres.ai/shared/components/AlertSnackbar/useAlertSnackbar";
import { AiBotProvider } from "./hooks";
import { useHideIntercom } from "../../hooks/useHideIntercom";

export interface BotWrapperProps {
  envData: {
    info?: {
      user_name?: string
    }
  };
  orgData: {
    id: number
  },
  history: RouteComponentProps['history']
  project?: string
  match: {
    params: {
      org?: string
      threadId?: string
    }
  }
}


export const BotWrapper = (props: BotWrapperProps) => {
  useHideIntercom();
  return (
    <AlertSnackbarProvider>
      <AiBotProvider args={{ threadId: props.match.params.threadId, orgId: props.orgData.id }}>
        <BotPage {...props} />
      </AiBotProvider>
    </AlertSnackbarProvider>
  )
}
