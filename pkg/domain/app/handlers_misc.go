package app

import (
	"legocerthub-backend/pkg/output"
	"legocerthub-backend/pkg/storage/sqlite"
	"net/http"

	"go.uber.org/zap/zapcore"
)

type appStatus struct {
	Status             string `json:"status"`
	ShowDebugInfo      bool   `json:"show_debug_info"`
	Version            string `json:"version"`
	DbUserVersion      int    `json:"database_version"`
	ConfigVersionMatch bool   `json:"config_version_match"`
}

// statusHandler writes some basic info about the status of the Application
func (app *Application) statusHandler(w http.ResponseWriter, r *http.Request) (err error) {
	cfgVerMatch := false
	if app.config.ConfigVersion != nil && *app.config.ConfigVersion == configVersion {
		cfgVerMatch = true
	}

	currentStatus := appStatus{
		Status:             "available",
		ShowDebugInfo:      app.logger.Level() == zapcore.DebugLevel,
		Version:            appVersion,
		DbUserVersion:      sqlite.DbCurrentUserVersion,
		ConfigVersionMatch: cfgVerMatch,
	}

	err = app.output.WriteJSON(w, http.StatusOK, currentStatus, "server")
	if err != nil {
		return err
	}

	return nil
}

// healthHandler writes some basic info about the status of the Application
func (app *Application) healthHandler(w http.ResponseWriter, r *http.Request) (err error) {
	// write 204 (No Content)
	app.output.WriteEmptyResponse(w, http.StatusNoContent)

	return nil
}

// notFoundHandler is called when there is not a matching route on the router
func (app *Application) notFoundHandler(w http.ResponseWriter, r *http.Request) (err error) {
	// OPTIONS should always return a response to prevent preflight errors
	// see: https://stackoverflow.com/questions/52047548/response-for-preflight-does-not-have-http-ok-status-in-angular
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	// return 404 not found
	err = app.output.WriteErrorJSON(w, output.ErrNotFound)
	if err != nil {
		return err
	}

	return nil
}

// redirectToFrontendRoot is a handler that redirects to the frontend app
func redirectToFrontendRoot(w http.ResponseWriter, r *http.Request) error {
	http.Redirect(w, r, frontendUrlPath, http.StatusPermanentRedirect)
	return nil
}
