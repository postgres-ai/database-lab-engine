import React from "react";
import { SyntaxHighlight } from "@postgres.ai/shared/components/SyntaxHighlight";

type DebugLogsProps = {
  isLoading: boolean
  isEmpty: boolean
  id: string
}

export const DebugLogs = (props: DebugLogsProps) => {
  const { isLoading, isEmpty, id } = props;
  return (
    <SyntaxHighlight
      id={`logs-container-${id}`}
      style={{
        overflowY: 'auto',
        height: '100%',
        overflowX: 'hidden',
        backgroundColor: 'rgb(250, 250, 250)',
        margin: 0,
      }}
      content={
        isLoading ?
          'Loading...'
          : isEmpty
            ? 'No debug information available.'
            : ''
      }
    />
  )
}