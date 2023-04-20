import CloseIcon from '@material-ui/icons/Close'
import { useState, useEffect, useCallback } from 'react'
import { makeStyles, Snackbar } from '@material-ui/core'
import CheckCircleIcon from '@material-ui/icons/CheckCircle'
import { Button } from '@postgres.ai/shared/components/MenuButton'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { capitalizeFirstLetter } from './utils'

import { getBillingStatus } from 'api/configs/getBillingStatus'
import { activateBilling } from 'api/configs/activateBilling'

import styles from './styles.module.scss'

const AUTO_HIDE_DURATION = 1500

export const StickyTopBar = () => {
  const useStyles = makeStyles(
    {
      errorNotification: {
        '& > div.MuiSnackbarContent-root': {
          backgroundColor: '#f44336!important',
          minWidth: '100%',
        },

        '&  div.MuiSnackbarContent-message': {
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '4px 0',
          gap: '10px',
          fontSize: '13px',
        },
      },
      successNotification: {
        '& > div.MuiSnackbarContent-root': {
          backgroundColor: '#4caf50!important',
          minWidth: '100%',
        },

        '& div.MuiSnackbarContent-message': {
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '4px 0',
          gap: '10px',
          fontSize: '13px',
        },
      },
    },
    { index: 1 },
  )

  const classes = useStyles()
  const [isLoading, setIsLoading] = useState(false)
  const [snackbarState, setSnackbarState] = useState<{
    isOpen: boolean
    message: string | null
    type: 'error' | 'success' | null
  }>({
    isOpen: false,
    message: null,
    type: null,
  })
  const [state, setState] = useState<{
    type: 'billingInactive' | 'missingOrgKey' | 'noConnection' | null
    pageUrl?: string
    message: string
  }>({
    type: null,
    pageUrl: '',
    message: '',
  })

  const handleReset = useCallback(() => {
    setState({
      type: null,
      message: '',
    })
  }, [])

  const handleResetSnackbar = useCallback(() => {
    setTimeout(() => {
      setSnackbarState({
        isOpen: false,
        message: null,
        type: null,
      })
    }, AUTO_HIDE_DURATION)
  }, [])

  const handleActivate = useCallback(() => {
    setIsLoading(true)
    activateBilling()
      .then((res) => {
        setIsLoading(false)
        if (res.response) {
          handleReset()
          setSnackbarState({
            isOpen: true,
            message: 'All DLE SE features are now active.',
            type: 'success',
          })
        } else {
          setSnackbarState({
            isOpen: true,
            message: capitalizeFirstLetter(res.error.message),
            type: 'error',
          })
        }
        handleResetSnackbar()
      })
      .finally(() => {
        setIsLoading(false)
      })
  }, [setIsLoading, handleReset, handleResetSnackbar])

  useEffect(() => {
    getBillingStatus().then((res) => {
      if (!res.response?.billing_active && res.response?.recognized_org) {
        setState({
          type: 'billingInactive',
          pageUrl: res.response?.recognized_org.billing_page,
          message:
            'No active payment methods are found for your organization on the Postgres.ai Platform; please, visit the',
        })
      } else if (!res.response?.recognized_org) {
        setState({
          type: 'missingOrgKey',
          message: capitalizeFirstLetter(res.error.message),
        })
      }
    })
  }, [handleResetSnackbar])

  useEffect(() => {
    const handleNoConnection = () => {
      setState({
        type: 'noConnection',
        message: 'No internet connection',
      })
    }

    const handleConnectionRestored = () => {
      setState({
        type: null,
        message: '',
      })
      handleActivate()
    }

    window.addEventListener('offline', handleNoConnection)
    window.addEventListener('online', handleConnectionRestored)

    return () => {
      window.removeEventListener('offline', handleNoConnection)
      window.removeEventListener('online', handleConnectionRestored)
    }
  }, [handleActivate])

  return (
    <>
      {state.type && (
        <div className={styles.container}>
          <p>{state.message}</p>
          &nbsp;
          {state.type === 'billingInactive' ? (
            <>
              <a href={state.pageUrl} target="_blank" rel="noreferrer">
                billing page
              </a>
              .&nbsp;Once resolved, &nbsp;
              <Button
                type="button"
                className={styles.activateBtn}
                onClick={handleActivate}
                disabled={isLoading}
              >
                re-activate DLE
                {isLoading && <Spinner size="sm" className={styles.spinner} />}
              </Button>
            </>
          ) : state.type === 'missingOrgKey' ? (
            <>
              Once resolved,&nbsp;
              <Button
                type="button"
                className={styles.activateBtn}
                onClick={handleActivate}
                disabled={isLoading}
              >
                re-activate DLE
                {isLoading && <Spinner size="sm" className={styles.spinner} />}
              </Button>
            </>
          ) : null}
        </div>
      )}
      <Snackbar
        open={snackbarState.isOpen}
        className={
          snackbarState.type === 'error'
            ? classes.errorNotification
            : snackbarState.type === 'success'
            ? classes.successNotification
            : ''
        }
        autoHideDuration={AUTO_HIDE_DURATION}
        message={
          <>
            {snackbarState.type === 'error' ? (
              <CloseIcon />
            ) : (
              snackbarState.type === 'success' && <CheckCircleIcon />
            )}
            {snackbarState.message}
          </>
        }
      />
    </>
  )
}
