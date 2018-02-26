package cmd

import "testing"

func TestHetznerConfig_FindSSHKeyByName(t *testing.T) {
	config := HetznerConfig{
		SSHKeys: []SSHKey{
			SSHKey{Name: "test-key1"},
			SSHKey{Name: "test-key2"},
		},
	}
	tests := []struct {
		name string
	}{
		{"test-key1"},
		{"non-existing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			idx, _ := config.FindSSHKeyByName(test.name)

			// I know this can be done more general, but this one is fast
			switch test.name {
			case "test-key1", "test-key2":
				if idx < 0 {
					t.Errorf("SSH key %s exists but not found", test.name)
				}
			default:
				if idx > -1 {
					t.Errorf("SSH key %s not exists but found", test.name)
				}
			}
		})
	}
}

func TestAppConfig_FindContextByName(t *testing.T) {
	config := getAppConfig()
	tests := []struct {
		name string
	}{
		{"first-context"},
		{"non-existing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := config.FindContextByName(test.name)

			switch test.name {
			case "first-context", "second-context":
				if err != nil {
					t.Errorf("unexpected error for context %s", test.name)
				}
			default:
				if err == nil {
					t.Errorf("no error for non-existing context %s", test.name)
				}
			}
		})
	}
}
func TestAppConfig_SwitchContextByName(t *testing.T) {
	config := getAppConfig()

	config.SwitchContextByName("second-context")

	if config.CurrentContext.Name != "second-context" {
		t.Error("could not switch context")
	}
}

func getAppConfig() AppConfig {
	config := AppConfig{
		Config: &HetznerConfig{
			Contexts: []HetznerContext{
				{Name: "first-context"},
				{Name: "second-context"},
			},
		},
	}
	return config
}


