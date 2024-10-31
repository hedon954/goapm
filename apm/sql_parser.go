package apm

import (
	"fmt"

	"github.com/xwb1989/sqlparser"
)

// sqlParser is a parser for sql statements.
type sqlParser struct{}

var SQLParser = &sqlParser{}

// parseTable parses the table name from the sql statement.
// If the sql statement is a multi-table statement, it returns true and we would ignore it in the following metrics.
func (p *sqlParser) parseTable(sql string) (tableName string, queryType int, multiTable bool, err error) {
	queryType = sqlparser.Preview(sql)
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return "", 0, false, fmt.Errorf("parse sql error: %w, sql: %s", err, sql)
	}

	switch queryType {
	case sqlparser.StmtInsert:
		t := stmt.(*sqlparser.Insert).Table.Name
		return t.CompliantName(), sqlparser.INSERT, false, nil
	case sqlparser.StmtDelete:
		tExprs := stmt.(*sqlparser.Delete).TableExprs
		if len(tExprs) > 1 {
			return "", 0, true, nil
		}
		t := sqlparser.GetTableName(tExprs[0].(*sqlparser.AliasedTableExpr).Expr)
		return t.CompliantName(), sqlparser.DELETE, false, nil
	case sqlparser.StmtUpdate:
		tExprs := stmt.(*sqlparser.Update).TableExprs
		if len(tExprs) > 1 {
			return "", 0, true, nil
		}
		t := sqlparser.GetTableName(tExprs[0].(*sqlparser.AliasedTableExpr).Expr)
		return t.CompliantName(), sqlparser.UPDATE, false, nil
	case sqlparser.StmtSelect:
		tExprs := stmt.(*sqlparser.Select).From
		if len(tExprs) > 1 {
			return "", 0, true, nil
		}
		t := sqlparser.GetTableName(tExprs[0].(*sqlparser.AliasedTableExpr).Expr)
		return t.CompliantName(), sqlparser.SELECT, false, nil
	}

	return "", 0, false, fmt.Errorf("unsupported sql type: %d, sql: %s", queryType, sql)
}
