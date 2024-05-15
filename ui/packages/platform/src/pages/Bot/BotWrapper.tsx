import { BotPage } from "./index";
import {RouteComponentProps} from "react-router";
import {AlertSnackbarProvider} from "@postgres.ai/shared/components/AlertSnackbar/useAlertSnackbar";

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
  return (
    <AlertSnackbarProvider>
      <BotPage {...props} />
    </AlertSnackbarProvider>
  )
}
