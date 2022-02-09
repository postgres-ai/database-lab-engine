package main

import (
	"fmt"
	"log"

	pg_query "github.com/pganalyze/pg_query_go/v2"
)

const idxExample = `
CREATE UNIQUE INDEX title_idx ON films (title);

ALTER TABLE pgbench_accounts
    ADD COLUMN test integer NOT NULL DEFAULT 0;
`

var stmts []*pg_query.RawStmt

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

	stmts = idxTree.GetStmts()

	processStmts(stmts)

	idxTree.Stmts = stmts

	fmt.Printf("Parse Tree after processing:\n%#v\n\n", idxTree.GetStmts()[1].Stmt.Node)

	resIdxStr, err := pg_query.Deparse(idxTree)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Optimized query:\n%v\n", resIdxStr)
	// CREATE UNIQUE INDEX CONCURRENTLY title_idx ON films USING btree (title);
	// ALTER TABLE pgbench_accounts ADD COLUMN test int;
	// ALTER TABLE pgbench_accounts ALTER COLUMN test SET DEFAULT 0;
}

func processStmts(stmts []*pg_query.RawStmt) {
	for _, stmt := range stmts {
		detectNodeType(stmt.Stmt)
	}
}

func detectNodeType(node *pg_query.Node) {
	switch stmt := node.Node.(type) {
	case *pg_query.Node_IndexStmt:
		IndexStmt(stmt)

	case *pg_query.Node_AlterTableStmt:
		fmt.Println("Alter Type")
		AlterStmt(stmt)

	case *pg_query.Node_SelectStmt:
		fmt.Println("Select Type")

	}
}

func IndexStmt(stmt *pg_query.Node_IndexStmt) {
	stmt.IndexStmt.Concurrent = true
}

func AlterStmt(stmt *pg_query.Node_AlterTableStmt) {
	initialCommands := stmt.AlterTableStmt.GetCmds()

	commands := []*pg_query.Node{}
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
					cmd.Node = v
					commands = append(commands, cmd)

/*					newCmd := &pg_query.Node{
						Node: &pg_query.Node_AlterTableCmd{
							AlterTableCmd: &pg_query.AlterTableCmd{
								Subtype:  pg_query.AlterTableType_AT_ColumnDefault,
								Name:     "",
								Num:      v.AlterTableCmd.GetNum(),
								Newowner: v.AlterTableCmd.GetNewowner(),
								Def: pg_query.MakeSimpleColumnDefNode(
									def.GetColname(), def.GetTypeName(), []*pg_query.Node{constraints[index]}, def.GetLocation()),
								Behavior:  0,
								MissingOk: false,
							},
						},
					}*/

	/*				newStmt := &pg_query.RawStmt{
						Stmt: &pg_query.Node{
							Node: &pg_query.Node_AlterTableStmt{
								AlterTableStmt: &pg_query.AlterTableStmt{
									Relation:  stmt.AlterTableStmt.GetRelation(),
									Cmds:      []*pg_query.Node{newCmd},
									Relkind:   0,
									MissingOk: false,
								},
							},
						},
						StmtLocation: 0,
						StmtLen:      0,
					}*/

					setDef := fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %v;`,
						stmt.AlterTableStmt.GetRelation().GetRelname(), def.Colname,
						constraints[index].GetConstraint().GetRawExpr().GetAConst().GetVal().GetInteger().GetIval())
						fmt.Println(setDef)

					scanDef, err := pg_query.Parse(setDef)
					if err != nil {
						log.Fatal(err)
					}

					stmts = append(stmts, scanDef.Stmts...)

					// TODO: Update rows

					// TODO: apply the rest constraints
					constraints = append(constraints[:index], constraints[index+1:]...)

					//commands = append(commands, newCmd)
				}

			default:
				cmd.Node = v
			}

		default:
			commands = append(commands, cmd)

			fmt.Printf("%T\n", v)
		}
	}

	stmt.AlterTableStmt.Cmds = commands
}
