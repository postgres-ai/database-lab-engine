/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { observer } from 'mobx-react-lite'
import { makeStyles } from '@material-ui/core'
import copy from 'copy-to-clipboard'

import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { ArrowDropDownIcon } from '@postgres.ai/shared/icons/ArrowDropDown'
import { Modal as ModalBase } from '@postgres.ai/shared/components/Modal'
import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import {
  Table,
  TableHead,
  TableRow,
  TableBody,
  TableHeaderCell,
  TableBodyCell,
  TableBodyCellMenu,
} from '@postgres.ai/shared/components/Table'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { isSameDayUTC, formatDateWithDistance } from '@postgres.ai/shared/utils/date'

import { Tags } from '@postgres.ai/shared/pages/Instance/components/Tags'
import { ModalReloadButton } from '@postgres.ai/shared/pages/Instance/components/ModalReloadButton'
import { getTags } from '../Snapshots/components/SnapshotsModal/utils'

const useStyles = makeStyles(
  {
    root: {
      fontSize: '14px',
      marginTop: 0,
      display: 'block',
    },
    container: {
      maxHeight: '400px',
    },
    cellContentCentered: {
      display: 'flex',
      alignItems: 'center',
    },
    sortIcon: {
      marginLeft: '8px',
      width: '10px',
    },
    emptyStub: {
      marginTop: '16px',
    },
  },
  { index: 1 },
)

export const SnapshotsModal = observer(() => {
  const classes = useStyles()
  const stores = useStores()

  const { snapshots } = stores.main
  if (!snapshots.data) return null

  const filteredSnapshots = snapshots.data.filter((snapshot) => {
    const isMatchedByDate =
      !stores.snapshotsModal.date ||
      isSameDayUTC(snapshot.dataStateAtDate, stores.snapshotsModal.date)

    const isMatchedByPool =
      !stores.snapshotsModal.pool ||
      snapshot.pool === stores.snapshotsModal.pool

    return isMatchedByDate && isMatchedByPool
  })

  const isEmpty = !filteredSnapshots.length

  return (
    <ModalBase
      isOpen={stores.snapshotsModal.isOpenModal}
      onClose={stores.snapshotsModal.closeModal}
      title={`Snapshots (${filteredSnapshots.length})`}
      classes={{ content: classes.root }}
      size="md"
      titleRightContent={
        <ModalReloadButton
          isReloading={stores.main.snapshots.isLoading}
          onReload={stores.main.reloadSnapshots}
        />
      }
      headerContent={
        <Tags
          data={getTags({
            date: stores.snapshotsModal.date,
            pool: stores.snapshotsModal.pool,
          })}
        />
      }
    >
      {isEmpty && <p className={classes.emptyStub}>No snapshots found</p>}

      {!isEmpty && (
        <HorizontalScrollContainer classes={{ content: classes.container }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell />
                <TableHeaderCell>Data state time</TableHeaderCell>
                <TableHeaderCell>
                  <div className={classes.cellContentCentered}>
                    Created
                    <ArrowDropDownIcon className={classes.sortIcon} />
                  </div>
                </TableHeaderCell>
                <TableHeaderCell>Disk</TableHeaderCell>
                <TableHeaderCell>Size</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredSnapshots.map((snapshot) => {
                return (
                  <TableRow key={snapshot.id}>
                    <TableBodyCellMenu
                      actions={[
                        {
                          name: 'Copy snapshot ID',
                          onClick: () => copy(snapshot.id),
                        },
                        {
                          name: 'Show related clones',
                          onClick: () =>
                            stores.clonesModal.openModal({
                              snapshotId: snapshot.id,
                            }),
                        },
                      ]}
                    />
                    <TableBodyCell>
                      {formatDateWithDistance(snapshot.dataStateAt, snapshot.dataStateAtDate)}
                    </TableBodyCell>
                    <TableBodyCell>
                      {formatDateWithDistance(snapshot.createdAt, snapshot.createdAtDate)}
                    </TableBodyCell>
                    <TableBodyCell>{snapshot.pool ?? '-'}</TableBodyCell>
                    <TableBodyCell>
                      {snapshot.physicalSize
                        ? formatBytesIEC(snapshot.physicalSize)
                        : '-'}
                    </TableBodyCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </HorizontalScrollContainer>
      )}
    </ModalBase>
  )
})
