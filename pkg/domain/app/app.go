package app

import (
	"legocerthub-backend/pkg/acme"
	"legocerthub-backend/pkg/domain/acme_accounts"
	"legocerthub-backend/pkg/domain/app/auth"
	"legocerthub-backend/pkg/domain/authorizations"
	"legocerthub-backend/pkg/domain/certificates"
	"legocerthub-backend/pkg/domain/orders"
	"legocerthub-backend/pkg/domain/private_keys"
	"legocerthub-backend/pkg/httpclient"
	"legocerthub-backend/pkg/output"
	"legocerthub-backend/pkg/storage/sqlite"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// application version
const appVersion = "0.1.0"

// Directory URLs for Let's Encrypt
const acmeProdUrl string = "https://acme-v02.api.letsencrypt.org/directory"
const acmeStagingUrl string = "https://acme-staging-v02.api.letsencrypt.org/directory"

// Application is the main app struct
type Application struct {
	config         *config
	logger         *zap.SugaredLogger
	appCert        *appCert
	httpClient     *httpclient.Client
	output         *output.Service
	router         *httprouter.Router
	storage        *sqlite.Storage
	auth           *auth.Service
	keys           *private_keys.Service
	acmeProd       *acme.Service
	acmeStaging    *acme.Service
	accounts       *acme_accounts.Service
	authorizations *authorizations.Service
	orders         *orders.Service
	certificates   *certificates.Service
}

// CloseStorage closes the storage connection
func (app *Application) CloseStorage() {
	app.storage.Close()
}

//

// return various app parts which are used as needed by services
func (app *Application) GetDevMode() bool {
	return *app.config.DevMode
}

func (app *Application) GetLogger() *zap.SugaredLogger {
	return app.logger
}

func (app *Application) GetHttpClient() *httpclient.Client {
	return app.httpClient
}

func (app *Application) GetOutputter() *output.Service {
	return app.output
}

// hacky workaround for storage since can't just combine into one interface
func (app *Application) GetAuthStorage() auth.Storage {
	return app.storage
}
func (app *Application) GetKeyStorage() private_keys.Storage {
	return app.storage
}
func (app *Application) GetAccountStorage() acme_accounts.Storage {
	return app.storage
}
func (app *Application) GetCertificatesStorage() certificates.Storage {
	return app.storage
}
func (app *Application) GetOrderStorage() orders.Storage {
	return app.storage
}

//

func (app *Application) GetKeysService() *private_keys.Service {
	return app.keys
}

func (app *Application) GetAcmeProdService() *acme.Service {
	return app.acmeProd
}

func (app *Application) GetAcmeStagingService() *acme.Service {
	return app.acmeStaging
}

func (app *Application) GetAcctsService() *acme_accounts.Service {
	return app.accounts
}

func (app *Application) GetAuthsService() *authorizations.Service {
	return app.authorizations
}
