package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// CHANGES v1 to v2:
// - certificates:
//     - Delete 'challenge_method' field/column

// renameOldDbV0Tables renames the existing tables to append _old to the end
// in preparation for re-creation of new tables.
func renameOldDbV1Tables(tx *sql.Tx) error {
	// rename tables
	query := `
		ALTER TABLE acme_accounts RENAME TO acme_accounts_old;
		ALTER TABLE acme_orders RENAME TO acme_orders_old;
		ALTER TABLE acme_servers RENAME TO acme_servers_old;
		ALTER TABLE certificates RENAME TO certificates_old;
		ALTER TABLE private_keys RENAME TO private_keys_old;
		ALTER TABLE users RENAME TO users_old;
	`

	_, err := tx.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

// removeOldDbV0Tables drops all of the _old table variants as a cleanup step
func removeOldDbV1Tables(tx *sql.Tx) error {
	// drop tables
	query := `
		DROP TABLE acme_orders_old;	
		DROP TABLE certificates_old;
		DROP TABLE acme_accounts_old;
		DROP TABLE private_keys_old;
		DROP TABLE acme_servers_old;
		DROP TABLE users_old;
	`

	_, err := tx.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

// createDBTablesV2 creates a fresh set of tables in the db using schema version 2
func createDBTablesV2(tx *sql.Tx) error {
	// acme_servers
	query := `CREATE TABLE IF NOT EXISTS acme_servers (
		id integer PRIMARY KEY AUTOINCREMENT NOT NULL UNIQUE,
		name text NOT NULL UNIQUE COLLATE NOCASE,
		description text NOT NULL,
		directory_url text NOT NULL UNIQUE,
		is_staging integer NOT NULL DEFAULT 0 CHECK(is_staging IN (0,1)),
		created_at integer NOT NULL,
		updated_at integer NOT NULL
	)`

	_, err := tx.Exec(query)
	if err != nil {
		return err
	}

	// private_keys
	query = `CREATE TABLE IF NOT EXISTS private_keys (
		id integer PRIMARY KEY AUTOINCREMENT NOT NULL UNIQUE,
		name text NOT NULL UNIQUE COLLATE NOCASE,
		description text NOT NULL,
		algorithm text NOT NULL,
		pem text NOT NULL UNIQUE,
		api_key text NOT NULL,
		api_key_new text NOT NULL DEFAULT '',
		api_key_disabled integer NOT NULL DEFAULT 0 CHECK(api_key_disabled IN (0,1)),
		api_key_via_url integer NOT NULL DEFAULT 0 CHECK(api_key_via_url IN (0,1)),
		created_at integer NOT NULL,
		updated_at integer NOT NULL
	)`

	_, err = tx.Exec(query)
	if err != nil {
		return err
	}

	// acme_accounts
	query = `CREATE TABLE IF NOT EXISTS acme_accounts (
		id integer PRIMARY KEY AUTOINCREMENT NOT NULL UNIQUE,
		name text NOT NULL UNIQUE COLLATE NOCASE,
		private_key_id integer NOT NULL UNIQUE,
		description text NOT NULL,
		status text NOT NULL DEFAULT 'unknown',
		email text NOT NULL,
		accepted_tos integer NOT NULL DEFAULT 0 CHECK(accepted_tos IN (0,1)),
		created_at integer NOT NULL,
		updated_at integer NOT NULL,
		kid text NOT NULL,
		acme_server_id integer NOT NULL,
		FOREIGN KEY (private_key_id)
			REFERENCES private_keys (id)
				ON DELETE RESTRICT
				ON UPDATE NO ACTION,
		FOREIGN KEY (acme_server_id)
			REFERENCES acme_servers (id)
				ON DELETE RESTRICT
				ON UPDATE NO ACTION
	)`

	_, err = tx.Exec(query)
	if err != nil {
		return err
	}

	// certificates
	query = `CREATE TABLE IF NOT EXISTS certificates (
		id integer PRIMARY KEY AUTOINCREMENT NOT NULL UNIQUE,
		private_key_id integer NOT NULL UNIQUE,
		acme_account_id integer NOT NULL,
		name text NOT NULL UNIQUE COLLATE NOCASE,
		description text NOT NULL,
		subject text NOT NULL,
		subject_alts text NOT NULL,
		csr_org text NOT NULL,
		csr_ou text NOT NULL,
		csr_country text NOT NULL,
		csr_state text NOT NULL,
		csr_city text NOT NULL,
		api_key text NOT NULL,
		api_key_new text NOT NULL DEFAULT '',
		api_key_via_url integer NOT NULL DEFAULT 0 CHECK(api_key_via_url IN (0,1)),
		created_at integer NOT NULL,
		updated_at integer NOT NULL,
		FOREIGN KEY (private_key_id)
			REFERENCES private_keys (id)
				ON DELETE RESTRICT
				ON UPDATE NO ACTION,
		FOREIGN KEY (acme_account_id)
			REFERENCES acme_accounts (id)
				ON DELETE RESTRICT
				ON UPDATE NO ACTION
	)`

	_, err = tx.Exec(query)
	if err != nil {
		return err
	}

	// ACME orders
	query = `CREATE TABLE IF NOT EXISTS acme_orders (
			id integer PRIMARY KEY AUTOINCREMENT NOT NULL UNIQUE,
			acme_account_id integer NOT NULL,
			certificate_id integer NOT NULL,
			acme_location text NOT NULL UNIQUE,
			status text NOT NULL,
			known_revoked integer NOT NULL DEFAULT 0 CHECK(known_revoked IN (0,1)),
			error text,
			expires integer,
			dns_identifiers text NOT NULL,
			authorizations text NOT NULL,
			finalize text NOT NULL,
			finalized_key_id integer,
			certificate_url text,
			pem text,
			valid_from integer,
			valid_to integer,
			created_at integer NOT NULL,
			updated_at integer NOT NULL,
			FOREIGN KEY (acme_account_id)
				REFERENCES acme_accounts (id)
					ON DELETE CASCADE
					ON UPDATE NO ACTION,
			FOREIGN KEY (finalized_key_id)
				REFERENCES private_keys (id)
					ON DELETE SET NULL
					ON UPDATE NO ACTION,
			FOREIGN KEY (certificate_id)
				REFERENCES certificates (id)
					ON DELETE CASCADE
					ON UPDATE NO ACTION
		)`

	_, err = tx.Exec(query)
	if err != nil {
		return err
	}

	// users (for login to app)
	query = `CREATE TABLE IF NOT EXISTS users (
		id integer PRIMARY KEY AUTOINCREMENT NOT NULL UNIQUE,
		username text NOT NULL UNIQUE,
		password_hash NOT NULL,
		created_at integer NOT NULL,
		updated_at integer NOT NULL
	)`

	_, err = tx.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

// migrateV1toV2 updates the storage db from user_version 1 to user_version 2, if it cannot
// do so, an error is returned and modification is aborted
func (store *Storage) migrateV1toV2() (int, error) {
	oldSchemaVer := 1
	newSchemaVer := 2

	store.logger.Infof("updating database user_version from %d to %d", oldSchemaVer, newSchemaVer)

	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()

	// create sql transaction to roll back in the event an error occurs
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	// verify correct current ver
	query := `PRAGMA user_version`
	row := tx.QueryRowContext(ctx, query)
	fileUserVersion := -1
	err = row.Scan(
		&fileUserVersion,
	)
	if err != nil {
		return -1, err
	}
	if fileUserVersion != oldSchemaVer {
		return -1, fmt.Errorf("cannot update db schema, current version %d (expected %d)", fileUserVersion, oldSchemaVer)
	}

	// rename old data tables
	err = renameOldDbV1Tables(tx)
	if err != nil {
		return -1, err
	}

	// create new tables
	err = createDBTablesV2(tx)
	if err != nil {
		return -1, err
	}

	// copy data from _old tables to new tables
	query = `
	INSERT INTO users SELECT * FROM users_old;
	INSERT INTO private_keys SELECT * FROM private_keys_old;
	INSERT INTO acme_servers SELECT * FROM acme_servers_old;
	INSERT INTO acme_accounts SELECT * FROM acme_accounts_old;
	`

	_, err = tx.Exec(query)
	if err != nil {
		return -1, err
	}

	// copy data from certificates_old (removing method column/field)
	query = `
	INSERT INTO certificates
	SELECT id, private_key_id, acme_account_id, name, description, subject, subject_alts, csr_org, csr_ou,
		csr_country, csr_state, csr_city, api_key, api_key_new, api_key_via_url, created_at, updated_at
	FROM certificates_old;
	`

	_, err = tx.Exec(query)
	if err != nil {
		return -1, err
	}

	// copy data from remaining table
	query = `
	INSERT INTO acme_orders SELECT * FROM acme_orders_old;
	`

	_, err = tx.Exec(query)
	if err != nil {
		return -1, err
	}

	// remove _old tables
	err = removeOldDbV1Tables(tx)
	if err != nil {
		return -1, err
	}

	// update user_version to 2
	query = fmt.Sprintf(`
		PRAGMA user_version = %d
	`, newSchemaVer)

	_, err = tx.Exec(query)
	if err != nil {
		return -1, err
	}

	// no errors, commit transaction
	err = tx.Commit()
	if err != nil {
		return -1, err
	}

	store.logger.Infof("database user_version successfully upgraded from %d to %d", oldSchemaVer, newSchemaVer)
	return newSchemaVer, nil
}
