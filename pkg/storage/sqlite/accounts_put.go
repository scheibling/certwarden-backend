package sqlite

import (
	"context"
	"errors"
	"legocerthub-backend/pkg/acme"
	"legocerthub-backend/pkg/domain/acme_accounts"
)

// accountPayloadToDb turns the client payload into a db object
func nameDescAccountPayloadToDb(payload acme_accounts.NameDescPayload) (accountDb, error) {
	var dbObj accountDb

	// mandatory, error if somehow does not exist
	if payload.ID == nil {
		return accountDb{}, errors.New("accounts: name/desc payload: missing id")
	}
	dbObj.id = intToNullInt32(payload.ID)

	// mandatory, error if somehow does not exist
	if payload.Name == nil {
		return accountDb{}, errors.New("accounts: name/desc payload: missing name")
	}
	dbObj.name = stringToNullString(payload.Name)

	dbObj.description = stringToNullString(payload.Description)

	return dbObj, nil
}

// putExistingAccountNameDesc only updates the name and desc in the database
// refactor to more generic for anything that can be updated??
func (store *Storage) PutNameDescAccount(payload acme_accounts.NameDescPayload) (err error) {
	// Load payload into db obj
	accountDb, err := nameDescAccountPayloadToDb(payload)
	if err != nil {
		return err
	}

	// database update
	ctx, cancel := context.WithTimeout(context.Background(), store.Timeout)
	defer cancel()

	query := `
	UPDATE
		acme_accounts
	SET
		name = $1,
		description = $2
	WHERE
		id = $3
	`

	_, err = store.Db.ExecContext(ctx, query,
		accountDb.name,
		accountDb.description,
		accountDb.id,
	)

	if err != nil {
		return err
	}

	return nil
}

// leAccountResponseToDb translates the ACME account response into the fields we want to save
// in the database
func leAccountResponseToDb(id int, response acme.AcmeAccountResponse) accountDb {
	var account accountDb
	var err error

	account.id = intToNullInt32(&id)
	email := response.Email()
	account.email = stringToNullString(&email)
	account.status = stringToNullString(&response.Status)
	account.kid = stringToNullString(response.Location)

	createdAt, err := response.CreatedAtUnix()
	if err != nil {
		createdAt = 0
	}
	account.createdAt = intToNullInt32(&createdAt)

	account.updatedAt = timeNow()

	return account
}

// PutLEAccountResponse populates an account with data that is returned by LE when
//  an account is POSTed to
func (store *Storage) PutLEAccountResponse(id int, response acme.AcmeAccountResponse) error {
	// Load id and response into db obj
	accountDb := leAccountResponseToDb(id, response)

	ctx, cancel := context.WithTimeout(context.Background(), store.Timeout)
	defer cancel()

	query := `
	UPDATE
		acme_accounts
	SET
		status = $1,
		email = $2,
		created_at = $3,
		updated_at = $4,
		kid = case when $5 is null then kid else $5 end
	WHERE
		id = $6`

	_, err := store.Db.ExecContext(ctx, query,
		accountDb.status,
		accountDb.email,
		accountDb.createdAt,
		accountDb.updatedAt,
		accountDb.kid,
		accountDb.id)
	if err != nil {
		return err
	}

	// TODO: Handle 0 rows updated.
	return nil
}
