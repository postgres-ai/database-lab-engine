/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import cn from 'classnames'
import { useEffect, useState } from 'react'
import copy from 'copy-to-clipboard'
import { makeStyles } from '@material-ui/core'
import { useHistory } from 'react-router-dom'
import { formatDistanceToNowStrict } from 'date-fns'

import { isValidDate } from '@postgres.ai/shared/utils/date'
import { ArrowDropDownIcon } from '@postgres.ai/shared/icons/ArrowDropDown'
import { Branch } from '@postgres.ai/shared/types/api/endpoints/getBranches'
import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { useHost } from '@postgres.ai/shared/pages/Instance/context'
import {
  Table,
  TableHead,
  TableRow,
  TableBody,
  TableHeaderCell,
  TableBodyCell,
  TableBodyCellMenu,
} from '@postgres.ai/shared/components/Table'

import { DeleteBranchModal } from '../Modals/DeleteBranchModal'
import { DeleteBranch } from '@postgres.ai/shared/types/api/endpoints/deleteBranch'

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
    interactiveRow: {
      cursor: 'pointer',
    },
    verticalCentered: {
      display: 'flex',
      alignItems: 'center',
    },
    marginTop: {
      marginTop: '16px',
    },
    sortIconUp: {
      transform: 'rotate(180deg)',
    },
  },
  { index: 1 },
)

export const BranchesTable = ({
  branches,
  emptyTableText,
  deleteBranch,
  branchesRoute,
  reloadBranches,
}: {
  branches: Branch[]
  emptyTableText: string
  deleteBranch: DeleteBranch
  branchesRoute: string
  reloadBranches: () => void
}) => {
  const host = useHost()
  const history = useHistory()
  const classes = useStyles()

  const [state, setState] = useState({
    sortByParent: 'desc',
    branches: [] as Branch[],
  })
  const [branchId, setBranchId] = useState('')
  const [isOpenDestroyModal, setIsOpenDestroyModal] = useState(false)

  const handlesortByParent = () => {
    const sortByParent = state.sortByParent === 'desc' ? 'asc' : 'desc'

    const sortedBranches = [...state.branches].sort((a, b) => {
      if (sortByParent === 'asc') {
        return a.parent.localeCompare(b.parent)
      } else {
        return b.parent.localeCompare(a.parent)
      }
    })

    setState({
      sortByParent,
      branches: sortedBranches,
    })
  }

  useEffect(() => {
    setState({
      sortByParent: 'desc',
      branches: branches ?? [],
    })
  }, [branches])

  if (!state.branches.length) {
    return <p className={classes.marginTop}>{emptyTableText}</p>
  }

  return (
    <HorizontalScrollContainer>
      <Table>
        <TableHead>
          <TableRow>
            <TableHeaderCell />
            <TableHeaderCell>Branch</TableHeaderCell>
            <TableHeaderCell>
              <div
                onClick={handlesortByParent}
                className={cn(classes.interactiveRow, classes.verticalCentered)}
              >
                Parent
                <ArrowDropDownIcon
                  className={cn(
                    state.sortByParent === 'asc' && classes.sortIconUp,
                    classes.sortIcon,
                  )}
                />
              </div>
            </TableHeaderCell>
            <TableHeaderCell>Data state time</TableHeaderCell>
            <TableHeaderCell>Latest Snapshot ID</TableHeaderCell>
            <TableHeaderCell>Number of snapshots</TableHeaderCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {state.branches?.map((branch) => {
            const branchPagePath = host.routes.branch(branch.name)

            return (
              <TableRow
                key={branch.name}
                hover
                onClick={() => history.push(branchPagePath)}
                className={classes.pointerCursor}
              >
                <TableBodyCellMenu
                  actions={[
                    {
                      name: 'Copy snapshot ID',
                      onClick: () => copy(branch.snapshotID),
                    },
                    {
                      name: 'Destroy branch',
                      onClick: () => {
                        setBranchId(branch.name)
                        setIsOpenDestroyModal(true)
                      },
                    },
                  ]}
                />

                <TableBodyCell>{branch.name}</TableBodyCell>
                <TableBodyCell>{branch.parent}</TableBodyCell>
                <TableBodyCell>
                  {branch.dataStateAt} (
                  {isValidDate(new Date(branch.dataStateAt))
                    ? formatDistanceToNowStrict(new Date(branch.dataStateAt), {
                        addSuffix: true,
                      })
                    : '-'}
                  )
                </TableBodyCell>
                <TableBodyCell>{branch.snapshotID}</TableBodyCell>
                <TableBodyCell>{branch.numSnapshots}</TableBodyCell>
              </TableRow>
            )
          })}
        </TableBody>
        <DeleteBranchModal
          isOpen={isOpenDestroyModal}
          onClose={() => {
            setIsOpenDestroyModal(false)
            setBranchId('')
          }}
          deleteBranch={deleteBranch}
          branchName={branchId}
          instanceId={host.instanceId}
          afterSubmitClick={() => {
            reloadBranches()
            history.push(branchesRoute)
          }}
        />
      </Table>
    </HorizontalScrollContainer>
  )
}
