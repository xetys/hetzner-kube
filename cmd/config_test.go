package cmd

import (
	"testing"

	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

func TestHetznerConfig_FindSSHKeyByName(t *testing.T) {
	config := getCloudProvider()
	tests := []string{
		"test-key1",
		"non-existing",
	}
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, err := config.FindSSHKeyByName(test)

			// I know this can be done more general, but this one is fast
			switch test {
			case "test-key1", "test-key2":
				if err != nil {
					t.Errorf("SSH key %s exists but not found", test)
				}
			default:
				if err == nil {
					t.Errorf("SSH key %s not exists but found", test)
				}
			}
		})
	}
}

func TestHetznerConfig_AddSSHKey(t *testing.T) {
	config := getCloudProvider()

	config.AddSSHKey(clustermanager.SSHKey{Name: "test-key3"})

	if len(config.SSHKeys) != 3 {
		t.Errorf("After adding SSH key size seems not valid")
	}
}

func TestHetznerConfig_DeleteSSHKey(t *testing.T) {
	config := getCloudProvider()

	config.DeleteSSHKey("test-key1")

	if len(config.SSHKeys) != 1 {
		t.Errorf("After removing SSH key size seems not valid")
	}
}

func TestHetznerConfig_DeleteNonExistingSSHKey(t *testing.T) {
	config := getCloudProvider()

	err := config.DeleteSSHKey("non-existing")

	if err == nil {
		t.Errorf("After removing non existing SSH key we should receive an error")
	}
}

func TestAppConfig_FindContextByName(t *testing.T) {
	config := getAppConfig()
	tests := []string{
		"first-context",
		"non-existing",
	}
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, err := config.FindContextByName(test)

			switch test {
			case "first-context", "second-context":
				if err != nil {
					t.Errorf("unexpected error for context %s", test)
				}
			default:
				if err == nil {
					t.Errorf("no error for non-existing context %s", test)
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
	return AppConfig{
		Config: &HetznerConfig{
			Contexts: []HetznerContext{
				{Name: "first-context"},
				{Name: "second-context"},
			},
		},
	}
}

func getCloudProvider() HetznerConfig {
	return HetznerConfig{
		SSHKeys: []clustermanager.SSHKey{
			{Name: "test-key1"},
			{Name: "test-key2"},
		},
	}
}
