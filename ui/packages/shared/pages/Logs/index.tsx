import copy from 'copy-to-clipboard'
import React, { useEffect, useReducer } from 'react'
import { Alert, AlertTitle } from '@material-ui/lab'
import { FormControlLabel, Checkbox, makeStyles } from '@material-ui/core'

import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { Api } from '@postgres.ai/shared/pages/Instance/stores/Main'
import { establishConnection } from '@postgres.ai/shared/pages/Logs/wsLogs'
import { useWsScroll } from '@postgres.ai/shared/pages/Logs/hooks/useWsScroll'
import { CopyToClipboard } from '@postgres.ai/shared/icons/CopyToClipboard'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { LAPTOP_WIDTH_PX } from '@postgres.ai/shared/pages/Logs/constants'

const useStyles = makeStyles(
  () => ({
    spinnerContainer: {
      display: 'flex',
      width: '100%',
      alignItems: 'center',
      justifyContent: 'center',
    },
    copyToClipboard: {
      position: 'fixed',
      right: '80px',
      cursor: 'pointer',
      zIndex: 1,
      backgroundColor: 'rgb(232, 244, 253)',
      borderRadius: '4px',
      padding: '4px 4px 0px 4px',
    },
    filterSection: {
      marginTop: '10px',
    },
    //  we need important since id has higher priority than class
    logsContainer: {
      overflow: 'auto !important',
      margin: '10px 0 0 0 !important',
      maxHeight: 'calc(100vh - 360px)',
      position: 'relative',

      '& p': {
        fontSize: '10px !important',
        maxWidth: 'calc(100% - 25px)',

        '@media (max-width: 982px)': {
          maxWidth: '100%',
        },
      },
    },
  }),
  { index: 1 },
)

export const Logs = ({ api }: { api: Api }) => {
  const initialState = {
    '[DEBUG]': true,
    '[INFO]': true,
    '[ERROR]': true,
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
      default:
        throw new Error()
    }
  }

  const classes = useStyles()
  const [isLoading, setIsLoading] = React.useState(true)
  const [state, dispatch] = useReducer(reducer, initialState)

  useWsScroll(isLoading)

  const targetNode = document.getElementById('logs-container')

  const handleCopyLogs = () => {
    targetNode && copy(targetNode?.innerText)
  }

  const FormCheckbox = ({ type }: { type: string }) => {
    return (
      <FormControlLabel
        control={
          <Checkbox
            checked={(state as Record<string, boolean>)[`[${type}]`]}
            onChange={() => dispatch({ type: type })}
            name={type}
            disabled={isLoading}
          />
        }
        label={type}
      />
    )
  }

  useEffect(() => {
    if (api.initWS != undefined) {
      establishConnection(api)
    }
  }, [api])

  useEffect(() => {
    localStorage.setItem('logsState', JSON.stringify(state))
  }, [state])

  useEffect(() => {
    const config = { attributes: false, childList: true, subtree: true }

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
  }, [isLoading, targetNode])

  return (
    <>
      <Alert severity="info">
        <AlertTitle>Sensitive values are masked.</AlertTitle>
        You can see the raw log data connecting to the machine and running{' '}
        <strong>'docker logs --since 5m -f dblab_server'</strong>.
      </Alert>
      {window.innerWidth > LAPTOP_WIDTH_PX && (
        <section className={classes.filterSection}>
          <SectionTitle level={2} tag="h2" text="Filters:" />
          <div>
            <FormCheckbox type="DEBUG" />
            <FormCheckbox type="INFO" />
            <FormCheckbox type="ERROR" />
          </div>
        </section>
      )}
      <div id="logs-container" className={classes.logsContainer}>
        {isLoading ? (
          <div className={classes.spinnerContainer}>
            <Spinner />
          </div>
        ) : window.innerWidth > LAPTOP_WIDTH_PX ? (
          <div className={classes.copyToClipboard} onClick={handleCopyLogs}>
            <Tooltip interactive content={'Copy logs to the clipboard'}>
              <CopyToClipboard />
            </Tooltip>
          </div>
        ) : null}
      </div>
    </>
  )
}
