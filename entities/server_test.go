package entities

import "testing"

// The state of an installation travels as a PascalCase string, and the
// spellings are fixed by the contract, not by the SDK.
func TestApplicationInstallationClosedSetsOnTheWire(t *testing.T) {
	fixed := map[string]struct {
		got  string
		want string
	}{
		"ApplicationInstallStateInstalling":           {string(ApplicationInstallStateInstalling), "Installing"},
		"ApplicationInstallStateInstalled":            {string(ApplicationInstallStateInstalled), "Installed"},
		"ApplicationInstallStateFailed":               {string(ApplicationInstallStateFailed), "Failed"},
		"ApplicationOutcomeReasonRegistrationFailed":  {string(ApplicationOutcomeReasonRegistrationFailed), "RegistrationFailed"},
		"ApplicationOutcomeReasonDockerNotReady":      {string(ApplicationOutcomeReasonDockerNotReady), "DockerNotReady"},
		"ApplicationOutcomeReasonServiceCreateFailed": {string(ApplicationOutcomeReasonServiceCreateFailed), "ServiceCreateFailed"},
		"ApplicationOutcomeReasonDeployTimeout":       {string(ApplicationOutcomeReasonDeployTimeout), "DeployTimeout"},
		"ApplicationOutcomeReasonHostUnreachable":     {string(ApplicationOutcomeReasonHostUnreachable), "HostUnreachable"},
		"ApplicationOutcomeReasonLLMKeyValueMissing":  {string(ApplicationOutcomeReasonLLMKeyValueMissing), "LlmKeyValueMissing"},
		"ApplicationComponentKindWeb":                 {string(ApplicationComponentKindWeb), "Web"},
		"ApplicationComponentKindWorker":              {string(ApplicationComponentKindWorker), "Worker"},
		"ApplicationComponentKindDatabase":            {string(ApplicationComponentKindDatabase), "Database"},
	}
	for name, c := range fixed {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
}

// An ordered application is refused before the request is built, and the message
// names the position in the list — that is the only way the caller can tell
// which of the applications it has to fix.
func TestCreateServerRequestValidateApplications(t *testing.T) {
	base := func() CreateServerRequest {
		return CreateServerRequest{
			LocationID: "dc-1",
			ImageID:    "img-ubuntu-24",
			CPU:        2,
			RamMB:      4096,
			Name:       "srv",
			Volumes:    []VolumeSpec{{Name: "boot", SizeMB: 40960}},
		}
	}

	cases := map[string]struct {
		applications []ApplicationSpec
		wantErr      string
	}{
		"no applications at all": {
			nil,
			"",
		},
		"application with parameters and a key": {
			[]ApplicationSpec{{
				ID:          "n8n",
				Parameters:  map[string]string{"N8N_HOST": "localhost"},
				IssueLLMKey: true,
			}},
			"",
		},
		"application without id": {
			[]ApplicationSpec{{Parameters: map[string]string{"N8N_HOST": "localhost"}}},
			"applications[0]: id is required",
		},
		"second application without id": {
			[]ApplicationSpec{{ID: "n8n"}, {}},
			"applications[1]: id is required",
		},
		"parameter without a name": {
			[]ApplicationSpec{{ID: "n8n", Parameters: map[string]string{"": "localhost"}}},
			"applications[0]: parameter name is required",
		},
		"second application's parameter without a name": {
			[]ApplicationSpec{
				{ID: "n8n"},
				{ID: "wordpress", Parameters: map[string]string{"": "localhost"}},
			},
			"applications[1]: parameter name is required",
		},
	}

	for name, tc := range cases {
		r := base()
		r.Applications = tc.applications

		err := r.Validate()
		switch {
		case tc.wantErr == "":
			if err != nil {
				t.Errorf("%s: Validate() = %v, want nil", name, err)
			}
		case err == nil:
			t.Errorf("%s: Validate() = nil, want %q", name, tc.wantErr)
		case err.Error() != tc.wantErr:
			t.Errorf("%s: Validate() = %q, want %q", name, err.Error(), tc.wantErr)
		}
	}
}
