package diff

import (
	pg_query "github.com/pganalyze/pg_query_go/v2"
)

func makeBeginTransactionStmt() *pg_query.Node {
	return &pg_query.Node{
		Node: &pg_query.Node_TransactionStmt{
			TransactionStmt: &pg_query.TransactionStmt{
				Kind: pg_query.TransactionStmtKind_TRANS_STMT_BEGIN,
			},
		},
	}
}

func makeCommitTransactionStmt() *pg_query.Node {
	return &pg_query.Node{
		Node: &pg_query.Node_TransactionStmt{
			TransactionStmt: &pg_query.TransactionStmt{
				Kind: pg_query.TransactionStmtKind_TRANS_STMT_COMMIT,
			},
		},
	}
}
