package cmd_test

import (
	"testing"

	"github.com/xetys/hetzner-kube/cmd"
	"github.com/xetys/hetzner-kube/pkg/clustermanager"
)

const (
	firstContext  string = "first-context"
	secondContext string = "second-context"
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
		firstContext,
		"non-existing",
	}
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, err := config.FindContextByName(test)

			switch test {
			case firstContext, secondContext:
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

	config.SwitchContextByName(secondContext)

	if config.CurrentContext.Name != secondContext {
		t.Error("could not switch context")
	}
}

func getAppConfig() cmd.AppConfig {
	return cmd.AppConfig{
		Config: &cmd.HetznerConfig{
			Contexts: []cmd.HetznerContext{
				{Name: firstContext},
				{Name: secondContext},
			},
		},
	}
}

func getCloudProvider() cmd.HetznerConfig {
	return cmd.HetznerConfig{
		SSHKeys: []clustermanager.SSHKey{
			{Name: "test-key1"},
			{Name: "test-key2"},
		},
	}
}
