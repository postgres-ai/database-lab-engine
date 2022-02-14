package diff

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

const idxExample = `
CREATE UNIQUE INDEX title_idx ON films (title);

DROP INDEX title_idx;

ALTER TABLE distributors 
	ADD CONSTRAINT zipchk CHECK (char_length(zipcode) = 5);

ALTER TABLE distributors 
	ADD CONSTRAINT distfk FOREIGN KEY (address) REFERENCES addresses (address);

ALTER TABLE pgbench_accounts
    ADD COLUMN test integer NOT NULL DEFAULT 0;
`

const expectedTpl = `CREATE UNIQUE INDEX CONCURRENTLY title_idx ON films USING btree (title);

DROP INDEX CONCURRENTLY title_idx;

ALTER TABLE distributors ADD CONSTRAINT zipchk CHECK (char_length(zipcode) = 5) NOT VALID;
BEGIN; ALTER TABLE distributors VALIDATE CONSTRAINT zipchk; COMMIT;

ALTER TABLE distributors ADD CONSTRAINT distfk FOREIGN KEY (address) REFERENCES addresses (address) NOT VALID;
BEGIN; ALTER TABLE distributors VALIDATE CONSTRAINT distfk; COMMIT;

ALTER TABLE pgbench_accounts ADD COLUMN test int;
ALTER TABLE pgbench_accounts ALTER COLUMN test SET DEFAULT 0`

var space = regexp.MustCompile(`\s+`)

func TestStatementParser(t *testing.T) {
	expected := space.ReplaceAllString(expectedTpl, " ")
	optimized, err := OptimizeQueries(idxExample)
	require.NoError(t, err)
	require.Equal(t, expected, optimized)
}
