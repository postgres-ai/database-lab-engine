// Package diff parses SQL queries and processes statements for optimization.
package diff

import (
	"fmt"
	"log"

	pg_query "github.com/pganalyze/pg_query_go/v2"
)

// OptimizeQueries rewrites incoming queries into queries with zero downtime risk.
func OptimizeQueries(queries string) (string, error) {
	idxTree, err := pg_query.Parse(queries)
	if err != nil {
		return "", fmt.Errorf("failed to parse queries %w", err)
	}

	log.Printf("Original query:\n%v\n\n", queries)
	log.Printf("Parse Tree:\n%#v\n\n", idxTree)

	stmts := idxTree.GetStmts()
	nodes := processStmts(stmts)
	idxTree.Stmts = nodes

	return pg_query.Deparse(idxTree)
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
		return AlterStmt(node)

	case *pg_query.Node_SelectStmt:
		fmt.Println("Select Type")
	}

	return wrapTransaction([]*pg_query.Node{node})
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

					alterStmts = append(alterStmts, wrapTransaction([]*pg_query.Node{node})...)

					defaultDefinitionTemp := fmt.Sprintf(`alter table %s alter column %s set default %v;`,
						alterTableStmt.GetRelation().GetRelname(), def.Colname,
						constraints[index].GetConstraint().GetRawExpr().GetAConst().GetVal().GetInteger().GetIval())

					alterStmts = append(alterStmts, wrapTransaction(generateNodes(defaultDefinitionTemp))...)

					// TODO: Update rows

					// TODO: apply the rest constraints
					constraints = append(constraints[:index], constraints[index+1:]...)
					fmt.Println(constraints)
				}

			case pg_query.AlterTableType_AT_AddConstraint:
				constraint := v.AlterTableCmd.Def.GetConstraint()
				constraint.SkipValidation = true

				alterStmts = append(alterStmts, wrapTransaction([]*pg_query.Node{node})...)

				validationTemp := fmt.Sprintf(`alter table %s validate constraint %s;`,
					alterTableStmt.GetRelation().GetRelname(), constraint.GetConname())

				alterStmts = append(alterStmts, wrapTransaction(generateNodes(validationTemp))...)

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

func generateNodes(nodeTemplate string) []*pg_query.Node {
	defDefinition, err := pg_query.Parse(nodeTemplate)
	if err != nil {
		log.Println(err)
		return nil
	}

	nodes := []*pg_query.Node{}
	for _, rawStmt := range defDefinition.Stmts {
		nodes = append(nodes, rawStmt.Stmt)
	}

	return nodes
}

// wrapTransaction wraps nodes into transaction statements.
func wrapTransaction(nodes []*pg_query.Node) []*pg_query.Node {
	begin := makeBeginTransactionStmt()
	commit := makeCommitTransactionStmt()

	nodes = append([]*pg_query.Node{begin}, nodes...)
	nodes = append(nodes, commit)

	return nodes
}
