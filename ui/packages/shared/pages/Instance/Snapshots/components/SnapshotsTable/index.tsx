/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { observer } from 'mobx-react-lite'
import { makeStyles } from '@material-ui/core'
import { formatDistanceToNowStrict } from 'date-fns'
import copy from 'copy-to-clipboard'
import { useHistory } from 'react-router-dom'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { generateSnapshotPageId } from '@postgres.ai/shared/pages/Instance/Snapshots/utils'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { ArrowDropDownIcon } from '@postgres.ai/shared/icons/ArrowDropDown'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'
import { isSameDayUTC, isValidDate } from '@postgres.ai/shared/utils/date'
import {
  Table,
  TableHead,
  TableRow,
  TableBody,
  TableHeaderCell,
  TableBodyCell,
  TableBodyCellMenu,
} from '@postgres.ai/shared/components/Table'

const useStyles = makeStyles(
  {
    cellContentCentered: {
      display: 'flex',
      alignItems: 'center',
    },
    pointerCursor: {
      cursor: 'pointer',
    },
    sortIcon: {
      marginLeft: '8px',
      width: '10px',
    },
  },
  { index: 1 },
)

export const SnapshotsTable = observer(() => {
  const history = useHistory()
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

  return (
    <HorizontalScrollContainer>
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
            <TableHeaderCell>Pool</TableHeaderCell>
            <TableHeaderCell>Number of clones</TableHeaderCell>
            <TableHeaderCell>Logical Size</TableHeaderCell>
            <TableHeaderCell>Physical Size</TableHeaderCell>
            <TableHeaderCell>Comment</TableHeaderCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {filteredSnapshots.map((snapshot) => {
            const snapshotPageId = generateSnapshotPageId(snapshot.id)
            return (
              <TableRow
                key={snapshot.id}
                hover
                onClick={() =>
                  snapshotPageId &&
                  history.push(`/instance/snapshots/${snapshotPageId}`)
                }
                className={classes.pointerCursor}
              >
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
                  {snapshot.dataStateAt}
                  {isValidDate(snapshot.dataStateAtDate)
                    ? formatDistanceToNowStrict(snapshot.dataStateAtDate, {
                        addSuffix: true,
                      })
                    : '-'}
                </TableBodyCell>
                <TableBodyCell>
                  {snapshot.createdAt} (
                  {isValidDate(snapshot.createdAtDate)
                    ? formatDistanceToNowStrict(snapshot.createdAtDate, {
                        addSuffix: true,
                      })
                    : '-'}
                  )
                </TableBodyCell>
                <TableBodyCell>{snapshot.pool ?? '-'}</TableBodyCell>
                <TableBodyCell>{snapshot.numClones ?? '-'}</TableBodyCell>
                <TableBodyCell>
                  {snapshot.physicalSize
                    ? formatBytesIEC(snapshot.logicalSize)
                    : '-'}
                </TableBodyCell>
                <TableBodyCell>
                  {snapshot.physicalSize
                    ? formatBytesIEC(snapshot.physicalSize)
                    : '-'}
                </TableBodyCell>
                <TableBodyCell>{snapshot.comment ?? '-'}</TableBodyCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </HorizontalScrollContainer>
  )
})
