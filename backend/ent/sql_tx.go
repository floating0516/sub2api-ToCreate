package ent

import (
	stdsql "database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
)

// SQLTx exposes the database/sql transaction used by this Ent transaction.
// It is intended for repositories that own tables outside the generated Ent schema.
func (tx *Tx) SQLTx() (*stdsql.Tx, error) {
	if tx == nil {
		return nil, fmt.Errorf("ent: nil transaction")
	}
	txDriver, ok := tx.config.driver.(*txDriver)
	if !ok {
		return nil, fmt.Errorf("ent: unexpected transaction driver %T", tx.config.driver)
	}
	dialectTx, ok := txDriver.tx.(*entsql.Tx)
	if !ok {
		return nil, fmt.Errorf("ent: unexpected dialect transaction %T", txDriver.tx)
	}
	sqlTx, ok := dialectTx.ExecQuerier.(*stdsql.Tx)
	if !ok {
		return nil, fmt.Errorf("ent: unexpected SQL transaction %T", dialectTx.ExecQuerier)
	}
	return sqlTx, nil
}
