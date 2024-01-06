package dns01manual

import (
	"errors"
	"legocerthub-backend/pkg/acme"
	"os/exec"
)

// Provision adds the corresponding DNS record using the script.
func (service *Service) Provision(domain, _, keyAuth string) error {
	// get dns record
	dnsRecordName, dnsRecordValue := acme.ValidationResourceDns01(domain, keyAuth)

	// run create script
	// script command
	cmd := service.makeCreateCommand(dnsRecordName, dnsRecordValue)

	// run script command
	result, err := cmd.Output()
	if err != nil {
		// try to get stderr and log it too
		exitErr := new(exec.ExitError)
		if errors.As(err, &exitErr) {
			service.logger.Errorf("acme.sh dns create script std err: %s", exitErr.Stderr)
		}

		service.logger.Errorf("dns create script error: %s", err)
		return err
	}
	service.logger.Debugf("dns create script output: %s", string(result))

	return nil
}

// Deprovision deletes the corresponding DNS record using the script.
func (service *Service) Deprovision(domain, _, keyAuth string) error {
	// get dns record
	dnsRecordName, dnsRecordValue := acme.ValidationResourceDns01(domain, keyAuth)

	// run delete script
	// script command
	cmd := service.makeDeleteCommand(dnsRecordName, dnsRecordValue)

	// run script command
	result, err := cmd.Output()
	if err != nil {
		// try to get stderr and log it too
		exitErr := new(exec.ExitError)
		if errors.As(err, &exitErr) {
			service.logger.Errorf("acme.sh dns create script std err: %s", exitErr.Stderr)
		}

		service.logger.Errorf("dns delete script error: %s", err)
		return err
	}
	service.logger.Debugf("dns delete script output: %s", string(result))

	return nil
}
