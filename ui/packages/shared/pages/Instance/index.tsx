/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useEffect } from 'react'
import { makeStyles } from '@material-ui/core'
import { observer } from 'mobx-react-lite'

import { Button } from '@postgres.ai/shared/components/Button2'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinner'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'

import { Tabs } from './Tabs'
import { Clones } from './Clones'
import { Info } from './Info'
import { ClonesModal } from './ClonesModal'
import { SnapshotsModal } from './SnapshotsModal'
import { HostProvider, StoresProvider, Host } from './context'

import { useCreatedStores } from './useCreatedStores'

type Props = Host

const useStyles = makeStyles((theme) => ({
  title: {
    marginTop: '8px',
  },
  reloadButton: {
    flex: '0 0 auto',
    alignSelf: 'flex-start',
  },
  errorStub: {
    marginTop: '16px',
  },
  content: {
    display: 'flex',
    marginTop: '16px',
    position: 'relative',
    flex: '1 1 100%',

    [theme.breakpoints.down('sm')]: {
      flexDirection: 'column',
    },
  },
}))

export const Instance = observer((props: Props) => {
  const classes = useStyles()

  const { instanceId } = props

  const stores = useCreatedStores(props)

  useEffect(() => {
    stores.main.load(instanceId)
  }, [instanceId])

  const { instance, instanceError } = stores.main

  useEffect(() => {
    if (instance && !instance?.state?.pools) {
      if (!props.callbacks) return

      props.callbacks.showDeprecatedApiBanner()
      return props.callbacks?.hideDeprecatedApiBanner
    }
  }, [instance])

  return (
    <HostProvider value={props}>
      <StoresProvider value={stores}>
        <>
          { props.elements.breadcrumbs }
          <SectionTitle
            text={props.title}
            level={1}
            tag="h1"
            className={classes.title}
            rightContent={
              <Button
                onClick={() => stores.main.load(props.instanceId)}
                isDisabled={!instance && !instanceError}
                className={classes.reloadButton}
              >
                Reload info
              </Button>
            }
          >
            <Tabs />
          </SectionTitle>

          {instanceError && (
            <ErrorStub {...instanceError} className={classes.errorStub} />
          )}

          {!instanceError && (
            <div className={classes.content}>
              {!instance && <StubSpinner />}

              {instance && (
                <>
                  <Clones />
                  <Info />
                </>
              )}
            </div>
          )}

          <ClonesModal />

          <SnapshotsModal />
        </>
      </StoresProvider>
    </HostProvider>
  )
})
