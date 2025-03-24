import React from "react";
import { Consulting } from "./index";
import { RouteComponentProps } from "react-router";

export interface ConsultingWrapperProps {
  orgId?: number;
  history: RouteComponentProps['history']
  project?: string
  match: {
    params: {
      org?: string
    }
  }
  orgData: {
    consulting_type: string | null
    alias: string
    role: {
      id: number
    }
  }
}

export const ConsultingWrapper = (props: ConsultingWrapperProps) => {
  return <Consulting {...props} />;
}