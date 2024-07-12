/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import { makeStyles, Theme, useMediaQuery } from '@material-ui/core'
import { useHistory } from 'react-router-dom'
import { observer } from 'mobx-react-lite'

import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { Button } from '@postgres.ai/shared/components/Button2'
import { round } from '@postgres.ai/shared/utils/numbers'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { InfoIcon } from '@postgres.ai/shared/icons/Info'

import { useStores, useHost } from '@postgres.ai/shared/pages/Instance/context'

import { ClonesList } from './ClonesList'
import { Header } from './Header'

const SHORT_LIST_SIZE = 3

interface ClonesProps {
  onlyRenderList?: boolean
}

const useStyles = makeStyles(
  (theme) => ({
    root: {
      width: 0,
      flex: '1 1 100%',
      marginRight: '40px',
      height: '100%',

      [theme.breakpoints.down('sm')]: {
        width: '100%',
        marginRight: 0,
      },
    },
    listSizeButton: {
      marginTop: '12px',
    },
    infoIcon: {
      height: '12px',
      width: '12px',
      marginLeft: '8px',
      color: '#808080',
    },
  }),
  { index: 1 },
)

export const Clones = observer((props: ClonesProps) => {
  const onlyRenderList = props?.onlyRenderList
  const classes = useStyles()
  const history = useHistory()
  const isMobile = useMediaQuery<Theme>((theme) => theme.breakpoints.down('sm'))
  const [isShortListForMobile, setIsShortListForMobile] = useState(true)

  const stores = useStores()
  const host = useHost()

  const { instance } = stores.main
  if (!instance || !instance.state) return null

  const isShortList = isMobile && isShortListForMobile && !onlyRenderList
  const toggleListSize = () => setIsShortListForMobile(!isShortListForMobile)

  const goToCloneAddPage = () => history.push(host.routes.createClone())

  const showListSizeButton =
    instance.state?.cloning.clones?.length > SHORT_LIST_SIZE && isMobile

  const isLoadingSnapshots = stores.main.snapshots.isLoading
  const hasSnapshots = Boolean(stores.main.snapshots.data?.length)
  const canCreateClone = hasSnapshots && !stores.main.isDisabledInstance

  return (
    <div className={classes.root}>
      {!onlyRenderList && (
        <>
          <SectionTitle level={2} tag="h2" text="Cloning summary" />
          <Header
            expectedCloningTimeS={round(
              instance.state.cloning.expectedCloningTime,
              2,
            )}
            logicalSize={instance.state.dataSize}
            clonesCount={instance.state.cloning.clones.length}
          />
        </>
      )}
      <SectionTitle
        level={2}
        tag="h3"
        text={`Clones (${instance.state.cloning.clones.length})`}
        rightContent={
          <>
            <Button
              theme="primary"
              onClick={goToCloneAddPage}
              isDisabled={!canCreateClone}
              isLoading={isLoadingSnapshots}
            >
              Create clone
            </Button>

            {!hasSnapshots && (
              <Tooltip content="No snapshots">
                <div style={{ display: 'flex' }}>
                  <InfoIcon className={classes.infoIcon} />
                </div>
              </Tooltip>
            )}
          </>
        }
      />

      <ClonesList
        clones={
          isShortList
            ? instance.state.cloning.clones.slice(0, SHORT_LIST_SIZE)
            : instance.state.cloning.clones
        }
        isDisabled={stores.main.isDisabledInstance}
        emptyStubText="This instance has no active clones"
      />

      {showListSizeButton && !onlyRenderList && (
        <Button className={classes.listSizeButton} onClick={toggleListSize}>
          {isShortList ? 'Show more' : 'Show less'}
        </Button>
      )}
    </div>
  )
})
