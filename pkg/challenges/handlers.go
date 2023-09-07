package challenges

import (
	"encoding/json"
	"legocerthub-backend/pkg/challenges/dns_checker"
	"legocerthub-backend/pkg/challenges/providers"
	"legocerthub-backend/pkg/output"
	"net/http"
)

// GetProvidersConfig returns all of the currently configured providers configs
func (service *Service) GetProvidersConfig(w http.ResponseWriter, r *http.Request) (err error) {
	err = service.output.WriteJSON(w, http.StatusOK, service.providers.Config(), "providers")
	if err != nil {
		return err
	}
	return nil
}

// SetProviders configures providers with the provided config
func (service *Service) SetProviders(w http.ResponseWriter, r *http.Request) (err error) {
	var cfg providers.Config

	// decode body into payload
	err = json.NewDecoder(r.Body).Decode(&cfg)
	if err != nil {
		service.logger.Debug(err)
		return output.ErrValidationFailed
	}

	// stop all existing providers
	err = service.providers.Stop()
	if err != nil {
		// if stop failed, its not possible to reliably recover
		service.logger.Fatalf("failed to stop challenge provider(s) (%s), fatal crash due to instability", err)
		// app exits
		return output.ErrInternal
	}

	// validation occurs through attempt to MakeProviders

	// create new providers with new configs
	ps, usesDns, makeErr := providers.MakeProviders(service.app, cfg)
	if makeErr != nil {
		// try to restart old providers
		err = service.providers.Start()
		if err != nil {
			service.logger.Fatalf("failed to restart previous challenge providers (%s), fatal crash due to instability", err)
			// app exits
			return output.ErrInternal
		}

		// restart success so app is stable, but update providers still failed
		service.logger.Debugf("failed to configure new challenge provider(s) (%s)", makeErr)
		return output.ErrValidationFailed
	}

	// success
	// if uses checker and not already running
	if usesDns && service.dnsChecker == nil {
		// enable checker
		service.logger.Info("new providers uses dns, enabling dns checker")
		service.dnsChecker, err = dns_checker.NewService(service.app, service.dnsCheckerCfg)
		if err != nil {
			sleepSecs := 120
			service.logger.Errorf("failed to configure dns checker (%s), attempting basic skip check and sleep %d secs config", err, sleepSecs)

			service.dnsChecker, err = dns_checker.NewService(service.app, dns_checker.Config{
				SkipCheckWaitSeconds: &sleepSecs,
			})
			if err != nil {
				service.logger.Error("failed to configure dns checker with sleep config (%s), reverting to previous providers", err)

				// try to restart old providers
				err = service.providers.Start()
				if err != nil {
					service.logger.Fatalf("failed to restart previous challenge providers (%s), fatal crash due to instability", err)
					// app exits
					return output.ErrInternal
				}
				return output.ErrInternal
			}
		}
	} else if !usesDns && service.dnsChecker != nil {
		// if not using dns and checker is running
		// remove dns checker service
		service.logger.Info("new providers does not use dns, disabling dns checker")
		service.dnsChecker = nil
	}

	// update service
	service.logger.Info("succesfully set new providers")
	service.providers = ps

	// return response to client
	response := output.JsonResponse{
		Status:  http.StatusOK,
		Message: "providers updated",
	}

	err = service.output.WriteJSON(w, response.Status, response, "response")
	if err != nil {
		return err
	}
	return nil
}
