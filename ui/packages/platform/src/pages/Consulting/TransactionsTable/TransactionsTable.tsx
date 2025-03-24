import React from "react";
import { Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from "@mui/material";
import { Transaction } from "stores/consulting";
import { formatPostgresInterval } from "../utils";
import { Link } from "@postgres.ai/shared/components/Link2";


type TransactionsTableProps = {
  transactions: Transaction[],
  alias: string
}

export const TransactionsTable = ({ transactions, alias }: TransactionsTableProps) => (
  <TableContainer component={Paper} sx={{ mt: 1 }}>
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>Action</TableCell>
          <TableCell>Amount</TableCell>
          <TableCell>Date</TableCell>
          <TableCell>Details</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {transactions.map(({ amount, created_at, issue_id, description, id }: Transaction) => (
          <TableRow key={id}>
            <TableCell sx={{ whiteSpace: 'nowrap' }}>{amount.charAt(0) === '-' ? 'Utilize' : 'Replenish'}</TableCell>
            <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatPostgresInterval(amount || '00')}</TableCell>
            <TableCell sx={{ whiteSpace: 'nowrap' }}>{new Date(created_at)?.toISOString()?.split('T')?.[0]}</TableCell>
            <TableCell>
              {issue_id ? (
                <Link external to={`https://gitlab.com/postgres-ai/postgresql-consulting/support/${alias}/-/issues/${issue_id}`} target="_blank">
                  {description}
                </Link>
              ) : description}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  </TableContainer>
);