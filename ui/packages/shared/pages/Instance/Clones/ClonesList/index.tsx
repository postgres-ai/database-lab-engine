/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import cn from 'classnames'
import { formatDistanceToNowStrict } from 'date-fns'
import { useHistory } from 'react-router-dom'

import {
  Table,
  TableHead,
  TableRow,
  TableHeaderCell,
  TableBody,
  TableBodyCell,
} from '@postgres.ai/shared/components/Table'
import { ArrowDropDownIcon } from '@postgres.ai/shared/icons/ArrowDropDown'
import { ShieldIcon } from '@postgres.ai/shared/icons/Shield'
import { RenewableIcon } from '@postgres.ai/shared/icons/Renewable'
import { InfoIcon } from '@postgres.ai/shared/icons/Info'
import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { Status } from '@postgres.ai/shared/components/Status'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'
import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { useHost } from '@postgres.ai/shared/pages/Instance/context'
import {
  getCloneStatusType,
  getCloneStatusText,
} from '@postgres.ai/shared/utils/clone'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { isValidDate } from '@postgres.ai/shared/utils/date'

import { MenuCell } from './MenuCell'
import { ConnectionModal } from './ConnectionModal'

import styles from './styles.module.scss'

type Props = {
  clones?: Clone[]
  isDisabled: boolean
  emptyStubText: string
  hideBranchingFeatures?: boolean
}

export const ClonesList = (props: Props) => {
  const host = useHost()
  const history = useHistory()

  const [state, setState] = useState({
    sortByBranch: 'desc',
    sortByCreated: 'desc',
    clones: props.clones ?? [],
  })

  const [cloneIdForConnect, setCloneIdForConnect] = useState<null | string>(
    null,
  )
  const [isOpenConnectionModal, setIsOpenConnectionModal] = useState(false)

  const openConnectionModal = (cloneId: string) => {
    setCloneIdForConnect(cloneId)
    setIsOpenConnectionModal(true)
  }

  const closeConnectionModal = () => {
    setIsOpenConnectionModal(false)
  }

  const handleSortByBranch = () => {
    const sortByBranch = state.sortByBranch === 'desc' ? 'asc' : 'desc'

    const sortedClones = [...state.clones].sort((a, b) => {
      if (sortByBranch === 'asc') {
        return a.branch.localeCompare(b.branch)
      } else {
        return b.branch.localeCompare(a.branch)
      }
    })

    setState({
      ...state,
      sortByBranch,
      clones: sortedClones,
    })
  }

  const handleSortByCreated = () => {
    const sortByCreated = state.sortByCreated === 'desc' ? 'asc' : 'desc'

    const sortedClones = [...state.clones].sort((a, b) => {
      if (sortByCreated === 'asc') {
        return (
          new Date(a.createdAtDate).getTime() -
          new Date(b.createdAtDate).getTime()
        )
      } else {
        return (
          new Date(b.createdAtDate).getTime() -
          new Date(a.createdAtDate).getTime()
        )
      }
    })

    setState({
      ...state,
      sortByCreated,
      clones: sortedClones,
    })
  }

  if (!state.clones?.length)
    return <p className={styles.emptyStub}>{props.emptyStubText}</p>

  return (
    <>
      <HorizontalScrollContainer>
        <Table>
          <TableHead>
            <TableRow>
              <TableHeaderCell />
              <TableHeaderCell>Status</TableHeaderCell>
              <TableHeaderCell>ID</TableHeaderCell>
              {!props.hideBranchingFeatures && <TableHeaderCell>
                <div
                  onClick={handleSortByBranch}
                  className={cn(styles.interactiveRow, styles.verticalCentered)}
                >
                  Branch
                  <ArrowDropDownIcon
                    className={cn(
                      state.sortByCreated === 'asc' && styles.hideSortIcon,
                      state.sortByBranch === 'asc' && styles.sortIconUp,
                      styles.sortIcon,
                    )}
                  />
                </div>
              </TableHeaderCell>}
              <TableHeaderCell>
                <Tooltip content="When enabled, neither manual nor automated deletion of this clone is possible. Note that abandoned protected clones may lead to out-of-disk-space events because they hold old data, blocking cleanup and refresh processes.">
                  <div className={styles.verticalCentered}>
                    Protected
                    <InfoIcon className={styles.infoIcon} />
                  </div>
                </Tooltip>
              </TableHeaderCell>
              <TableHeaderCell>
                <div
                  onClick={handleSortByCreated}
                  className={cn(styles.interactiveRow, styles.verticalCentered)}
                >
                  Created
                  <ArrowDropDownIcon
                    className={cn(
                      state.sortByBranch === 'asc' && styles.hideSortIcon,
                      state.sortByCreated === 'asc' && styles.sortIconUp,
                      styles.sortIcon,
                    )}
                  />
                </div>
              </TableHeaderCell>
              <TableHeaderCell>Port</TableHeaderCell>
              <TableHeaderCell>DB user</TableHeaderCell>
              <TableHeaderCell>
                <Tooltip content="Clone's own size – how much data was added or modified.">
                  <div className={styles.verticalCentered}>
                    Diff size
                    <InfoIcon className={styles.infoIcon} />
                  </div>
                </Tooltip>
              </TableHeaderCell>
              <TableHeaderCell>Disk</TableHeaderCell>
              <TableHeaderCell>Data state time</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {state.clones.map((clone) => {
              const clonePagePath = host.routes.clone(clone.id)

              return (
                <TableRow
                  hover={!props.isDisabled}
                  key={clone.id}
                  onClick={
                    props.isDisabled
                      ? undefined
                      : () => history.push(clonePagePath)
                  }
                  className={cn(!props.isDisabled && styles.interactiveRow)}
                >
                  <MenuCell
                    clone={clone}
                    onConnect={openConnectionModal}
                    clonePagePath={clonePagePath}
                  />
                  <TableBodyCell>
                    <Tooltip content={clone.status.message}>
                      <Status type={getCloneStatusType(clone.status.code)}>
                        {getCloneStatusText(clone.status.code)}
                      </Status>
                    </Tooltip>
                  </TableBodyCell>
                  <TableBodyCell>{clone.id}</TableBodyCell>
                  {!props.hideBranchingFeatures && <TableBodyCell>{clone.branch}</TableBodyCell>}
                  <TableBodyCell>
                    {clone.protected ? (
                      <Tooltip content="Clone is protected from manual and automated deletion. Note that abandoned protected clones may lead to out-of-disk-space events because they hold old data, blocking cleanup and refresh processes.">
                        <ShieldIcon className={styles.protectionIcon} />
                      </Tooltip>
                    ) : (
                      <Tooltip content="Clone is not protected from deletion. To save disk space it will be automatically deleted if there is no activity for a long time.">
                        <RenewableIcon className={styles.protectionIcon} />
                      </Tooltip>
                    )}
                  </TableBodyCell>
                  <TableBodyCell>
                    {clone.createdAt} (
                    {isValidDate(clone.createdAtDate)
                      ? formatDistanceToNowStrict(clone.createdAtDate, {
                          addSuffix: true,
                        })
                      : '-'}
                    )
                  </TableBodyCell>
                  <TableBodyCell>{clone.db.port}</TableBodyCell>
                  <TableBodyCell>{clone.db.username}</TableBodyCell>
                  <TableBodyCell>
                    {formatBytesIEC(clone.metadata.cloneDiffSize)}
                  </TableBodyCell>
                  <TableBodyCell>{clone.snapshot?.pool ?? '-'}</TableBodyCell>
                  <TableBodyCell>
                    {clone.snapshot ? (
                      <>
                        {clone.snapshot.dataStateAt} (
                        {isValidDate(clone.snapshot.dataStateAtDate)
                          ? formatDistanceToNowStrict(
                              clone.snapshot.dataStateAtDate,
                              {
                                addSuffix: true,
                              },
                            )
                          : '-'}
                        )
                      </>
                    ) : (
                      '-'
                    )}
                  </TableBodyCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </HorizontalScrollContainer>

      {cloneIdForConnect && (
        <ConnectionModal
          cloneId={cloneIdForConnect}
          isOpen={isOpenConnectionModal}
          onClose={closeConnectionModal}
        />
      )}
    </>
  )
}
