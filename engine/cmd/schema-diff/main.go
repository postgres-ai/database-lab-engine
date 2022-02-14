package main

import (
	"fmt"
	"log"

	"gitlab.com/postgres-ai/database-lab/v3/internal/schema/diff"
)

const idxExample = `
CREATE UNIQUE INDEX title_idx ON films (title);
`

/*
Optimized queries:

CREATE UNIQUE INDEX CONCURRENTLY title_idx ON films USING btree (title);
*/

func main() {
	resIdxStr, err := diff.OptimizeQueries(idxExample)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Optimized queries:\n%v\n", resIdxStr)
}
