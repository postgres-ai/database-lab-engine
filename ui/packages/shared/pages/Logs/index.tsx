import React, { useEffect } from 'react'
import { Alert, AlertTitle } from '@material-ui/lab'
import { makeStyles } from '@material-ui/core'
import { Spinner } from '@postgres.ai/shared/components/Spinner'

import { Api } from '@postgres.ai/shared/pages/Instance/stores/Main'
import { establishConnection } from '@postgres.ai/shared/pages/Logs/wsLogs'
import { useWsScroll } from '@postgres.ai/shared/pages/Logs/hooks/useWsScroll'

const useStyles = makeStyles(
  () => ({
    spinnerContainer: {
      display: 'flex',
      width: '100%',
      alignItems: 'center',
      justifyContent: 'center',
    },
  }),
  { index: 1 },
)

export const Logs = ({ api }: { api: Api }) => {
  const classes = useStyles()
  const [isLoading, setIsLoading] = React.useState(true)
  useWsScroll(isLoading)

  const logsFilterState =
    localStorage?.getItem('logsFilter') &&
    JSON?.parse(localStorage?.getItem('logsFilter') || '')

  const isEmpty = (obj: Record<string, boolean>) => {
    for (const key in obj) {
      if (obj.hasOwnProperty(key)) return false
    }
    return true
  }

  const initialState = {
    '[DEBUG]': !isEmpty(logsFilterState) ? logsFilterState?.['[DEBUG]'] : true,
    '[INFO]': !isEmpty(logsFilterState) ? logsFilterState?.['[INFO]'] : true,
    '[ERROR]': !isEmpty(logsFilterState) ? logsFilterState?.['[ERROR]'] : true,
    '[base.go]': !isEmpty(logsFilterState)
      ? logsFilterState?.['[base.go]']
      : true,
    '[runners.go]': !isEmpty(logsFilterState)
      ? logsFilterState?.['[runners.go]']
      : true,
    '[snapshots.go]': !isEmpty(logsFilterState)
      ? logsFilterState?.['[snapshots.go]']
      : true,
    '[util.go]': !isEmpty(logsFilterState)
      ? logsFilterState?.['[util.go]']
      : true,
    '[logging.go]': !isEmpty(logsFilterState)
      ? logsFilterState?.['[logging.go]']
      : false,
    '[ws.go]': !isEmpty(logsFilterState) ? logsFilterState?.['[ws.go]'] : false,
    '[other]': !isEmpty(logsFilterState) ? logsFilterState?.['[other]'] : true,
  }

  const reducer = (
    state: Record<string, boolean>,
    action: { type: string },
  ) => {
    switch (action.type) {
      case 'DEBUG':
        return { ...state, '[DEBUG]': !state['[DEBUG]'] }
      case 'INFO':
        return { ...state, '[INFO]': !state['[INFO]'] }
      case 'ERROR':
        return { ...state, '[ERROR]': !state['[ERROR]'] }
      case 'base.go':
        return { ...state, '[base.go]': !state['[base.go]'] }
      case 'runners.go':
        return { ...state, '[runners.go]': !state['[runners.go]'] }
      case 'snapshots.go':
        return { ...state, '[snapshots.go]': !state['[snapshots.go]'] }
      case 'logging.go':
        return { ...state, '[logging.go]': !state['[logging.go]'] }
      case 'util.go':
        return { ...state, '[util.go]': !state['[util.go]'] }
      case 'ws.go':
        return { ...state, '[ws.go]': !state['[ws.go]'] }
      case 'other':
        return { ...state, '[other]': !state['[other]'] }
      default:
        throw new Error()
    }
  }

  const [state, dispatch] = useReducer(reducer, initialState)

  const FormCheckbox = ({ type }: { type: string }) => {
    const filterType = (state as Record<string, boolean>)[`[${type}]`]
    return (
      <span
        onClick={() => dispatch({ type })}
        className={
          filterType && type !== 'ERROR'
            ? classes.activeButton
            : filterType && type === 'ERROR'
            ? classes.activeError
            : classes.passiveButton
        }
      >
        <span>{type.toLowerCase()}</span>
        <button aria-label="close" type="button">
          <PlusIcon />
        </button>
      </span>
    )
  }

  useEffect(() => {
    if (api.initWS != undefined) {
      establishConnection(api)
    }
  }, [api])

  useEffect(() => {
    const config = { attributes: false, childList: true, subtree: true }
    const targetNode = document.getElementById('logs-container') as HTMLElement

    if (isLoading && targetNode?.querySelectorAll('p').length === 1) {
      setIsLoading(false)
    }

    const callback = (mutationList: MutationRecord[]) => {
      for (const mutation of mutationList) {
        if (mutation.type === 'childList') {
          setIsLoading(false)
        }
      }
    }

    const observer = new MutationObserver(callback)
    targetNode && observer.observe(targetNode, config)
  }, [isLoading])

  return (
    <>
      <Alert severity="info">
        <AlertTitle>Sensitive data are masked.</AlertTitle>
        You can see the raw log data connecting to the machine and running the{' '}
        <strong>'docker logs'</strong> command.
      </Alert>
      <div id="logs-container">
        {isLoading && (
          <div className={classes.spinnerContainer}>
            <Spinner />
          </div>
        )}
      </div>
    </>
  )
}
