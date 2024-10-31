package apm

import "database/sql"

type dbUtils struct{}

var DBUtils = &dbUtils{}

func (d *dbUtils) Query(rows *sql.Rows, err any) []map[string]any {
	if err != nil {
		return nil
	}
	if rows == nil {
		return make([]map[string]any, 0)
	}
	defer func() { _ = rows.Close() }()

	columns, err := rows.Columns()
	if err != nil {
		return nil
	}
	scanArgs := make([]any, len(columns))
	values := make([]any, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	res := make([]map[string]any, 0)
	for rows.Next() {
		record := make(map[string]any)
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil
		}
		for i, col := range values {
			if col != nil {
				switch v := col.(type) {
				case []byte:
					record[columns[i]] = string(v)
				default:
					record[columns[i]] = col
				}
			}
		}

		res = append(res, record)
	}
	return res
}

func (d *dbUtils) QueryFirst(rows *sql.Rows, err any) map[string]any {
	res := d.Query(rows, err)
	if len(res) == 0 {
		return nil
	}
	return res[0]
}
