package main

import (
	"fmt"
	"log"

	pg_query "github.com/pganalyze/pg_query_go/v2"
)

const idxExample = `
CREATE UNIQUE INDEX title_idx ON films (title);

DROP INDEX title_idx;

ALTER TABLE distributors 
	ADD CONSTRAINT zipchk CHECK (char_length(zipcode) = 5);

ALTER TABLE pgbench_accounts
    ADD COLUMN test integer NOT NULL DEFAULT 0;
`

/*
Optimized queries:

CREATE UNIQUE INDEX CONCURRENTLY title_idx ON films USING btree (title);

DROP INDEX CONCURRENTLY title_idx;

ALTER TABLE distributors ADD CONSTRAINT zipchk CHECK (char_length(zipcode) = 5) NOT VALID;
ALTER TABLE distributors VALIDATE CONSTRAINT zipchk;

ALTER TABLE pgbench_accounts ADD COLUMN test int;
ALTER TABLE pgbench_accounts ALTER COLUMN test SET DEFAULT 0;
*/

func main() {
	scanTree, err := pg_query.ParseToJSON(idxExample)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("JSON: %s\n", scanTree)

	idxTree, err := pg_query.Parse(idxExample)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original query:\n%v\n\n", idxExample)
	fmt.Printf("Parse Tree:\n%#v\n\n", idxTree)

	stmts := idxTree.GetStmts()
	nodes := processStmts(stmts)
	idxTree.Stmts = nodes

	fmt.Printf("Parse Tree after processing:\n%#v\n\n", idxTree.GetStmts())

	resIdxStr, err := pg_query.Deparse(idxTree)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Optimized queries:\n%v\n", resIdxStr)
}

func processStmts(stmts []*pg_query.RawStmt) []*pg_query.RawStmt {
	rawStmts := []*pg_query.RawStmt{}

	for _, stmt := range stmts {
		for _, node := range detectNodeType(stmt.Stmt) {
			rawStmt := &pg_query.RawStmt{
				Stmt: node,
			}

			rawStmts = append(rawStmts, rawStmt)
		}
	}

	return rawStmts
}

func detectNodeType(node *pg_query.Node) []*pg_query.Node {
	switch stmt := node.Node.(type) {
	case *pg_query.Node_IndexStmt:
		IndexStmt(stmt)

	case *pg_query.Node_DropStmt:
		DropStmt(stmt)

	case *pg_query.Node_AlterTableStmt:
		fmt.Println("Alter Type")
		return AlterStmt(node)

	case *pg_query.Node_SelectStmt:
		fmt.Println("Select Type")
	}

	return []*pg_query.Node{node}
}

// IndexStmt processes index statement.
func IndexStmt(stmt *pg_query.Node_IndexStmt) {
	stmt.IndexStmt.Concurrent = true
}

// DropStmt processes drop statement.
func DropStmt(stmt *pg_query.Node_DropStmt) {
	switch stmt.DropStmt.RemoveType {
	case pg_query.ObjectType_OBJECT_INDEX:
		stmt.DropStmt.Concurrent = true
	default:
	}
}

// AlterStmt processes alter statement.
func AlterStmt(node *pg_query.Node) []*pg_query.Node {
	alterTableStmt := node.GetAlterTableStmt()
	if alterTableStmt == nil {
		return []*pg_query.Node{node}
	}

	var alterStmts []*pg_query.Node

	initialCommands := alterTableStmt.GetCmds()

	for _, cmd := range initialCommands {
		switch v := cmd.Node.(type) {
		case *pg_query.Node_AlterTableCmd:
			fmt.Printf("%#v\n", v)
			fmt.Printf("%#v\n", v.AlterTableCmd.Def.Node)
			fmt.Println(v.AlterTableCmd.Subtype.Enum())

			switch v.AlterTableCmd.Subtype {
			case pg_query.AlterTableType_AT_AddColumn:
				def := v.AlterTableCmd.Def.GetColumnDef()

				constraints := def.GetConstraints()
				constraintsMap := make(map[pg_query.ConstrType]int)

				for i, constr := range constraints {
					constraintsMap[constr.GetConstraint().Contype] = i
				}

				if index, ok := constraintsMap[pg_query.ConstrType_CONSTR_DEFAULT]; ok {
					def.Constraints = make([]*pg_query.Node, 0)

					alterStmts = append(alterStmts, node)

					defaultDefinitionTemp := fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %v;`,
						alterTableStmt.GetRelation().GetRelname(), def.Colname,
						constraints[index].GetConstraint().GetRawExpr().GetAConst().GetVal().GetInteger().GetIval())

					alterStmts = append(alterStmts, generateNode(defaultDefinitionTemp))

					// TODO: Update rows

					// TODO: apply the rest constraints
					constraints = append(constraints[:index], constraints[index+1:]...)
					fmt.Println(constraints)
				}

			case pg_query.AlterTableType_AT_AddConstraint:
				constraint := v.AlterTableCmd.Def.GetConstraint()
				constraint.SkipValidation = true

				alterStmts = append(alterStmts, node)

				validationTemp := fmt.Sprintf(`ALTER TABLE %s VALIDATE CONSTRAINT %s;`,
					alterTableStmt.GetRelation().GetRelname(), constraint.GetConname())

				alterStmts = append(alterStmts, generateNode(validationTemp))

			default:
				alterStmts = append(alterStmts, node)
			}

		default:
			alterStmts = append(alterStmts, node)

			fmt.Printf("%T\n", v)
		}
	}

	return alterStmts
}

func generateNode(nodeTemplate string) *pg_query.Node {
	defDefinition, err := pg_query.Parse(nodeTemplate)
	if err != nil {
		log.Fatal(err)
	}

	return defDefinition.Stmts[0].Stmt
}
