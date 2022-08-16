package logical

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/activity"
)

const psqlTestResult = "|||Activity|AutoVacuumMain\\\\postgres|||Activity|LogicalLauncherMain\\\\" +
	"john|0.083282|SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY|Client|ClientRead\\\\" +
	"john|6.941369|COPY public.pgbench_accounts (aid, bid, abalance, filler) TO stdout;|Client|ClientWrite\\\\" +
	"postgres|1.351549|COPY public.pgbench_accounts (aid, bid, abalance, filler) FROM\r\n stdin; ||\\\\" +
	"postgres|10.305525|COPY public.pgbench_accounts (aid, bid, abalance, filler) FRO\r\nM stdin; |Client|ClientRead"

func TestParsingStatActivity(t *testing.T) {
	expected := []activity.PGEvent{
		{
			User:          "",
			Query:         "",
			Duration:      0,
			WaitEventType: "Activity",
			WaitEvent:     "AutoVacuumMain",
		},
		{
			User:          "postgres",
			Query:         "",
			Duration:      0,
			WaitEventType: "Activity",
			WaitEvent:     "LogicalLauncherMain",
		},
		{
			User:          "john",
			Query:         "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY",
			Duration:      0.083282,
			WaitEventType: "Client",
			WaitEvent:     "ClientRead",
		},
		{
			User:          "john",
			Query:         "COPY public.pgbench_accounts (aid, bid, abalance, filler) TO stdout;",
			Duration:      6.941369,
			WaitEventType: "Client",
			WaitEvent:     "ClientWrite",
		},

		{
			User:          "postgres",
			Query:         "COPY public.pgbench_accounts (aid, bid, abalance, filler) FROM stdin; ",
			Duration:      1.351549,
			WaitEventType: "",
			WaitEvent:     "",
		},
		{
			User:          "postgres",
			Query:         "COPY public.pgbench_accounts (aid, bid, abalance, filler) FROM stdin; ",
			Duration:      10.305525,
			WaitEventType: "Client",
			WaitEvent:     "ClientRead",
		},
	}

	result := parseStatActivity(psqlTestResult)
	assert.Equal(t, expected, result)
}
