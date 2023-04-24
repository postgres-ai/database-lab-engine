import classNames from 'classnames'
import { makeStyles } from '@material-ui/core'
import { Alert, AlertTitle } from '@material-ui/lab'
import React, { useEffect, useReducer } from 'react'

import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { Api } from '@postgres.ai/shared/pages/Instance/stores/Main'
import { establishConnection } from '@postgres.ai/shared/pages/Logs/wsLogs'
import { useWsScroll } from '@postgres.ai/shared/pages/Logs/hooks/useWsScroll'
import { LAPTOP_WIDTH_PX } from '@postgres.ai/shared/pages/Logs/constants'
import { PlusIcon } from './Icons/PlusIcon'

const useStyles = makeStyles(
  () => ({
    spinnerContainer: {
      display: 'flex',
      width: '100%',
      alignItems: 'center',
      justifyContent: 'center',
    },
    filterSection: {
      marginTop: '10px',
      display: 'flex',
      flexDirection: 'row',
      gap: 10,

      '&  > span': {
        display: 'flex',
        flexDirection: 'row',
        gap: '5px',
        alignItems: 'center',
        border: '1px solid #898E9A',
        padding: '3px 8px',
        borderRadius: 5,
        fontSize: '13px',
        textTransform: 'capitalize',
        cursor: 'pointer',
      },

      '&  > span > button': {
        background: 'none',
        outline: 'none',
        border: 0,
        width: '18px',
        height: '18px',
        cursor: 'pointer',
      },
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
    activeButton: {
      border: '1px solid #3F51B5 !important',
      color: '#3F51B5 !important',

      '& svg': {
        '& path': {
          fill: '#3F51B5 !important',
        },
      },
    },
    passiveButton: {
      '& svg': {
        transform: 'rotate(45deg) scale(0.75)',
      },
    },
    activeError: {
      border: '1px solid #F44336 !important',
      color: '#F44336 !important',

      '& svg': {
        '& path': {
          fill: '#F44336 !important',
        },
      },
    },
    utilFilter: {
      '& > span': {
        textTransform: 'lowercase',
      },

      '& > span:last-child': {
        textTransform: 'capitalize',
      },
    },
  }),
  { index: 1 },
)

export const Logs = ({ api }: { api: Api }) => {
  const classes = useStyles()
  const [isLoading, setIsLoading] = React.useState(true)
  const targetNode = document.getElementById('logs-container')
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
    localStorage.setItem('logsFilter', JSON.stringify(state))
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
        <>
          <section className={classes.filterSection}>
            {Object.keys(state)
              .slice(0, 3)
              .map((key) => (
                <FormCheckbox
                  key={key}
                  type={key.replace('[', '').replace(']', '')}
                />
              ))}
          </section>
          <section
            className={classNames(classes.filterSection, classes.utilFilter)}
          >
            {Object.keys(state)
              .slice(3, 10)
              .map((key) => (
                <FormCheckbox
                  key={key}
                  type={key.replace('[', '').replace(']', '')}
                />
              ))}
          </section>
        </>
      )}
      <div id="logs-container" className={classes.logsContainer}>
        {isLoading ? (
          <div className={classes.spinnerContainer}>
            <Spinner />
          </div>
        ) : null}
      </div>
    </>
  )
}
