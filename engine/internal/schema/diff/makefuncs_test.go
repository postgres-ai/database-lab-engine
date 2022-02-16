package diff

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v2"
	"github.com/stretchr/testify/require"
)

func TestMakeFuncHelpers(t *testing.T) {
	t.Run("begin transaction statement", func(t *testing.T) {
		beginStmt := makeBeginTransactionStmt()

		transactionStmt := beginStmt.GetTransactionStmt()
		require.NotNil(t, transactionStmt)
		require.Equal(t, pg_query.TransactionStmtKind_TRANS_STMT_BEGIN, transactionStmt.GetKind())
	})

	t.Run("commit transaction statement", func(t *testing.T) {
		commitStmt := makeCommitTransactionStmt()

		transactionStmt := commitStmt.GetTransactionStmt()
		require.NotNil(t, transactionStmt)
		require.Equal(t, pg_query.TransactionStmtKind_TRANS_STMT_COMMIT, transactionStmt.GetKind())
	})
}
