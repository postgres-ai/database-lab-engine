/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import {
  Table as TableBase,
  TableProps as TableBaseProps,
  TableHead,
  TableRow,
  TableCell as TableCellBase,
  TableCellProps as TableCellBaseProps,
  TableBody,
  makeStyles,
} from '@material-ui/core'
import clsx from 'clsx'

import { colors } from '@postgres.ai/shared/styles/colors'

import { RowMenu, Props as RowMenuProps } from './RowMenu'

const cellStyles = {
  paddingLeft: '8px',
  paddingRight: '8px',
  borderColor: colors.consoleStroke,
}

// Table.
type TableProps = TableBaseProps

const useTableStyles = makeStyles(
  {
    root: {
      whiteSpace: 'nowrap',

      '& .MuiTableCell-root': {
        lineHeight: 'normal',
      },
    },
  },
  { index: 1 },
)

const Table = (props: TableProps) => {
  const classes = useTableStyles()
  return (
    <TableBase {...props} className={clsx(props.className, classes.root)} />
  )
}

// TableHeaderCell.
type TableHeaderCellProps = TableCellBaseProps

const useTableHeaderCellStyles = makeStyles(
  {
    root: {
      ...cellStyles,
      color: colors.pgaiDarkGray,
      paddingTop: '12px',
      paddingBottom: '12px',
    },
  },
  { index: 1 },
)

const TableHeaderCell = (props: TableHeaderCellProps) => {
  const classes = useTableHeaderCellStyles()

  return (
    <TableCellBase {...props} className={clsx(props.className, classes.root)} />
  )
}

// TableBodyCell.
type TableBodyCellProps = TableCellBaseProps

const useTableBodyCellStyles = makeStyles(
  {
    root: {
      ...cellStyles,
      fontSize: '12px',
      paddingTop: '8px',
      paddingBottom: '8px',
    },
  },
  { index: 1 },
)

const TableBodyCell = (props: TableBodyCellProps) => {
  const classes = useTableBodyCellStyles()

  return (
    <TableCellBase {...props} className={clsx(props.className, classes.root)} />
  )
}

// TableBodyCellMenu.
type TableBodyCellMenuProps = TableBodyCellProps & RowMenuProps

const useTableBodyCellMenuStyles = makeStyles(
  {
    root: {
      padding: 0,
    },
  },
  { index: 1 },
)

const TableBodyCellMenu = (props: TableBodyCellMenuProps) => {
  const classes = useTableBodyCellMenuStyles()

  const { isLoading, isDisabled, children, ...hiddenProps } = props

  const handleClick: React.MouseEventHandler<HTMLTableDataCellElement> = (e) =>
    e.stopPropagation()

  return (
    <TableBodyCell
      {...hiddenProps}
      className={classes.root}
      onClick={handleClick}
    >
      <RowMenu {...hiddenProps} isLoading={isLoading} isDisabled={isDisabled} />
      {children}
    </TableBodyCell>
  )
}

export {
  Table,
  TableHead,
  TableRow,
  TableBody,
  TableHeaderCell,
  TableBodyCell,
  TableBodyCellMenu,
}
