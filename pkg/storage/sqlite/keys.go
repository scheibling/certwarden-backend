package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"legocerthub-backend/pkg/private_keys"
	"legocerthub-backend/pkg/utils"
	"strconv"
	"time"
)

// a single private key, as database table fields
type keyDb struct {
	ID             int
	Name           string
	Description    sql.NullString
	AlgorithmValue string
	Pem            string
	ApiKey         string
	CreatedAt      int
	UpdatedAt      int
}

// KeyDbToKey translates the db object into the object the key service expects
func (keyDb *keyDb) keyDbToKey() private_keys.Key {
	return private_keys.Key{
		ID:          keyDb.ID,
		Name:        keyDb.Name,
		Description: keyDb.Description.String,
		Algorithm:   utils.AlgorithmByValue(keyDb.AlgorithmValue),
		Pem:         keyDb.Pem,
		ApiKey:      keyDb.ApiKey,
		CreatedAt:   keyDb.CreatedAt,
		UpdatedAt:   keyDb.UpdatedAt,
	}
}

// dbGetAllPrivateKeys writes information about all private keys to json
func (storage Storage) GetAllKeys() ([]private_keys.Key, error) {
	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `SELECT id, name, description, algorithm
	FROM private_keys ORDER BY id`

	rows, err := storage.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allKeys []private_keys.Key
	for rows.Next() {
		var oneKeyDb keyDb
		err = rows.Scan(
			&oneKeyDb.ID,
			&oneKeyDb.Name,
			&oneKeyDb.Description,
			&oneKeyDb.AlgorithmValue,
		)
		if err != nil {
			return nil, err
		}

		convertedKey := oneKeyDb.keyDbToKey()

		allKeys = append(allKeys, convertedKey)
	}

	return allKeys, nil
}

// dbGetOneKey returns a key from the db based on unique id
func (storage Storage) GetOneKey(id int) (private_keys.Key, error) {
	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `SELECT id, name, description, algorithm, pem, api_key, created_at, updated_at
	FROM private_keys
	WHERE id = $1
	ORDER BY id`

	row := storage.Db.QueryRowContext(ctx, query, id)

	var oneKeyDb keyDb
	err := row.Scan(
		&oneKeyDb.ID,
		&oneKeyDb.Name,
		&oneKeyDb.Description,
		&oneKeyDb.AlgorithmValue,
		&oneKeyDb.Pem,
		&oneKeyDb.ApiKey,
		&oneKeyDb.CreatedAt,
		&oneKeyDb.UpdatedAt,
	)

	if err != nil {
		return private_keys.Key{}, err
	}

	convertedKey := oneKeyDb.keyDbToKey()

	return convertedKey, nil
}

// dbPutExistingKey sets an existing key equal to the PUT values (overwriting
//  old values)
func (storage *Storage) PutExistingKey(payload private_keys.KeyPayload) error {
	// load fields that are permitted to be updated
	var key keyDb
	var err error

	key.ID, err = strconv.Atoi(payload.ID)
	if err != nil {
		return err
	}
	key.Name = payload.Name

	key.Description.Valid = true
	key.Description.String = payload.Description

	key.UpdatedAt = int(time.Now().Unix())

	// database action
	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `
	UPDATE
		private_keys
	SET
		name = $1,
		description = $2,
		updated_at = $3
	WHERE
		id = $4`

	_, err = storage.Db.ExecContext(ctx, query,
		key.Name,
		key.Description,
		key.UpdatedAt,
		key.ID)
	if err != nil {
		return err
	}

	// TODO: Handle 0 rows updated.

	return nil
}

// dbPostNewKey creates a new key based on what was POSTed
func (storage *Storage) PostNewKey(payload private_keys.KeyPayload) error {
	// load fields
	var key keyDb
	var err error

	key.Name = payload.Name

	key.Description.Valid = true
	key.Description.String = payload.Description

	key.AlgorithmValue = payload.AlgorithmValue
	key.Pem = payload.PemContent

	// generate api key
	key.ApiKey, err = utils.GenerateApiKey()
	if err != nil {
		return err
	}

	key.CreatedAt = int(time.Now().Unix())
	key.UpdatedAt = key.CreatedAt

	// database action
	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `
	INSERT INTO private_keys (name, description, algorithm, pem, api_key, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = storage.Db.ExecContext(ctx, query,
		key.Name,
		key.Description,
		key.AlgorithmValue,
		key.Pem,
		key.ApiKey,
		key.CreatedAt,
		key.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// delete a private key from the database
func (storage *Storage) DeleteKey(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `
	DELETE FROM
		private_keys
	WHERE
		id = $1
	`

	// TODO: Ensure can't delete a key that is in use on an account or certificate

	result, err := storage.Db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	resultRows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if resultRows == 0 {
		return errors.New("keys: Delete: failed to db delete -- 0 rows changed")
	}

	return nil
}
