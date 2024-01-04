/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

 import React from 'react'
 import cn from 'classnames'
 import { observer } from 'mobx-react-lite'
 import { makeStyles } from '@material-ui/core'
 import { formatDistanceToNowStrict } from 'date-fns'
 import copy from 'copy-to-clipboard'
 import { useHistory } from 'react-router-dom'
 
 import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
 import { generateSnapshotPageId } from '@postgres.ai/shared/pages/Instance/Snapshots/utils'
 import { DestroySnapshotModal } from '@postgres.ai/shared/pages/Snapshots/Snapshot/DestorySnapshotModal'
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
       cursor: 'pointer',
       transition: 'transform 0.15s ease-in-out',
     },
 
     sortIconUp: {
       transform: 'rotate(180deg)',
     },
 
     hideSortIcon: {
       opacity: 0,
     },
 
     verticalCentered: {
       display: 'flex',
       alignItems: 'center',
     },
   },
   { index: 1 },
 )
 
 export const SnapshotsTable = observer(() => {
   const history = useHistory()
   const classes = useStyles()
   const stores = useStores()
   const { snapshots } = stores.main
 
   const [snapshotModal, setSnapshotModal] = React.useState({
     isOpen: false,
     snapshotId: '',
   })
 
   const filteredSnapshots = snapshots?.data?.filter((snapshot) => {
     const isMatchedByDate =
       !stores.snapshotsModal.date ||
       isSameDayUTC(snapshot.dataStateAtDate, stores.snapshotsModal.date)
 
     const isMatchedByPool =
       !stores.snapshotsModal.pool ||
       snapshot.pool === stores.snapshotsModal.pool
 
     return isMatchedByDate && isMatchedByPool
   })
 
   const [state, setState] = React.useState({
     sortByCreatedDate: 'desc',
     snapshots: filteredSnapshots ?? [],
   })
 
   const handleSortByCreatedDate = () => {
     const sortByCreatedDate =
       state.sortByCreatedDate === 'desc' ? 'asc' : 'desc'
 
     const sortedSnapshots = [...state.snapshots].sort((a, b) => {
       if (sortByCreatedDate === 'asc') {
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
       sortByCreatedDate,
       snapshots: sortedSnapshots,
     })
   }
 
   if (!snapshots.data) return null
 
   return (
     <HorizontalScrollContainer>
       <Table>
         <TableHead>
           <TableRow>
             <TableHeaderCell />
             <TableHeaderCell>Data state time</TableHeaderCell>
             <TableHeaderCell>
               <div
                 className={cn(classes.pointerCursor, classes.verticalCentered)}
                 onClick={handleSortByCreatedDate}
               >
                 Created
                 <ArrowDropDownIcon
                   className={cn(
                     state.sortByCreatedDate === 'asc' && classes.sortIconUp,
                     classes.sortIcon,
                   )}
                 />
               </div>
             </TableHeaderCell>
             <TableHeaderCell>Pool</TableHeaderCell>
             <TableHeaderCell>Number of clones</TableHeaderCell>
             <TableHeaderCell>Logical Size</TableHeaderCell>
             <TableHeaderCell>Physical Size</TableHeaderCell>
           </TableRow>
         </TableHead>
         <TableBody>
           {state.snapshots?.map((snapshot) => {
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
                     {
                       name: 'Delete snapshot',
                       onClick: () =>
                         setSnapshotModal({
                           isOpen: true,
                           snapshotId: snapshot.id,
                         }),
                     },
                   ]}
                 />
                 <TableBodyCell>
                   {snapshot.dataStateAt} (
                   {isValidDate(snapshot.dataStateAtDate)
                     ? formatDistanceToNowStrict(snapshot.dataStateAtDate, {
                         addSuffix: true,
                       })
                     : '-'}
                   )
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
                   {snapshot.logicalSize
                     ? formatBytesIEC(snapshot.logicalSize)
                     : '-'}
                 </TableBodyCell>
                 <TableBodyCell>
                   {snapshot.physicalSize
                     ? formatBytesIEC(snapshot.physicalSize)
                     : '-'}
                 </TableBodyCell>
               </TableRow>
             )
           })}
         </TableBody>
         {snapshotModal.isOpen && snapshotModal.snapshotId && (
           <DestroySnapshotModal
             isOpen={snapshotModal.isOpen}
             onClose={() => setSnapshotModal({ isOpen: false, snapshotId: '' })}
             snapshotId={snapshotModal.snapshotId}
             afterSubmitClick={() =>
               stores.main?.reload(stores.main.instance?.id ?? '')
             }
           />
         )}
       </Table>
     </HorizontalScrollContainer>
   )
 })
 