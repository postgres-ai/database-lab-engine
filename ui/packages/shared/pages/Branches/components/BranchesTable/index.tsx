/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import copy from 'copy-to-clipboard'
import { makeStyles } from '@material-ui/core'
import { useHistory } from 'react-router-dom'

import { GetBranchesResponseType } from '@postgres.ai/shared/types/api/endpoints/getBranches'
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

import { DeleteBranchModal } from '../Modals/DeleteBranchModal'

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
    marginTop: {
      marginTop: '16px',
    },
  },
  { index: 1 },
)

export const BranchesTable = ({
  branchesData,
  emptyTableText,
  deleteBranch,
  deleteBranchError,
}: {
  branchesData: GetBranchesResponseType[]
  emptyTableText: string
  deleteBranch: (branchId: string) => void
  deleteBranchError: { title?: string; message?: string } | null
}) => {
  const history = useHistory()
  const classes = useStyles()

  const [branchId, setBranchId] = useState('')
  const [isOpenDestroyModal, setIsOpenDestroyModal] = useState(false)

  if (!branchesData.length) {
    return <p className={classes.marginTop}>{emptyTableText}</p>
  }

  return (
    <HorizontalScrollContainer>
      <Table>
        <TableHead>
          <TableRow>
            <TableHeaderCell />
            <TableHeaderCell>Branch</TableHeaderCell>
            <TableHeaderCell>Parent</TableHeaderCell>
            <TableHeaderCell>Data state time</TableHeaderCell>
            <TableHeaderCell>Snapshot ID</TableHeaderCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {branchesData?.map((branch) => (
            <TableRow
              key={branch.name}
              hover
              onClick={() => history.push(`/instance/branches/${branch.name}`)}
              className={classes.pointerCursor}
            >
              <TableBodyCellMenu
                actions={[
                  {
                    name: 'Copy snapshot ID',
                    onClick: () => copy(branch.snapshotID),
                  },
                  {
                    name: 'Delete branch',
                    onClick: () => {
                      setBranchId(branch.name)
                      setIsOpenDestroyModal(true)
                    },
                  },
                ]}
              />

              <TableBodyCell>{branch.name}</TableBodyCell>
              <TableBodyCell>{branch.parent}</TableBodyCell>
              <TableBodyCell>{branch.dataStateAt || '-'}</TableBodyCell>
              <TableBodyCell>{branch.snapshotID}</TableBodyCell>
            </TableRow>
          ))}
        </TableBody>
        <DeleteBranchModal
          isOpen={isOpenDestroyModal}
          onClose={() => {
            setIsOpenDestroyModal(false)
            setBranchId('')
          }}
          deleteBranchError={deleteBranchError}
          deleteBranch={deleteBranch}
          branchName={branchId}
        />
      </Table>
    </HorizontalScrollContainer>
  )
}
