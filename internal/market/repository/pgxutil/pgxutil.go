package pgxutil

import "github.com/jackc/pgx/v5"

// CollectAndConvert collects rows from pgx.Rows, scans each into Row, and converts to *Entity.
// For conversions that cannot fail.
func CollectAndConvert[Row, Entity interface{}](rows pgx.Rows, convertFn func(Row) *Entity) ([]*Entity, error) {
	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[Row])
	if err != nil {
		return nil, err
	}
	list := make([]*Entity, 0, len(slice))
	for _, row := range slice {
		list = append(list, convertFn(row))
	}
	return list, nil
}

// CollectAndConvertErr collects rows from pgx.Rows, scans each into Row, and converts to *Entity.
// For conversions that can fail.
func CollectAndConvertErr[Row, Entity interface{}](rows pgx.Rows, convertFn func(Row) (*Entity, error)) ([]*Entity, error) {
	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[Row])
	if err != nil {
		return nil, err
	}
	list := make([]*Entity, 0, len(slice))
	for _, row := range slice {
		item, err := convertFn(row)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, nil
}
