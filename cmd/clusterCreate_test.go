package cmd

import (
	"testing"
)

func TestClusterCmdValidate(t *testing.T) {

	AppConf.Config = &HetznerConfig{
		SSHKeys: []SSHKey{
			{Name:"test"},
		},
	}

	cmd := clusterCreateCmd


	cmd.ParseFlags([]string{"cluster", "create", "--ha-enabled", "--ssh-key", "test"})
	err := validateClusterCreateFlags(cmd, []string{})

	if err != nil {
		t.Error(err)
	}

	cmd.ParseFlags([]string{"cluster", "create", "--ha-enabled", "--ssh-key", "test", "--worker-count", "10"})
	err = validateClusterCreateFlags(cmd, []string{})

	if err != nil {
		t.Error(err)
	}

	cmd.ParseFlags([]string{"cluster", "create", "--ha-enabled", "--ssh-key", "test", "--worker-count", "0"})
	err = validateClusterCreateFlags(cmd, []string{})

	if err == nil {
		t.Error("no errors occurred with worker count 0, but should")
	}
	cmd.ParseFlags([]string{"cluster", "create", "--ha-enabled", "--ssh-key", "test", "--master-count", "1"})
	err = validateClusterCreateFlags(cmd, []string{})

	if err == nil {
		t.Error("no errors occurred with master count 1 in HA mode, but should")
	}

	cmd.ParseFlags([]string{"cluster", "create", "--ha-enabled", "--ssh-key", "test", "--master-count", "2", "--isolated-etcd", "--etcd-count", "2"})
	err = validateClusterCreateFlags(cmd, []string{})

	if err == nil {
		t.Error("no errors occurred with etcd count 2 in HA mode, but should")
	}

	cmd.ParseFlags([]string{"cluster", "create", "--ha-enabled", "--ssh-key", "test", "--master-count", "2",  "--etcd-count", "3"})
	err = validateClusterCreateFlags(cmd, []string{})

	if err == nil {
		t.Error("no errors occurred with provided etcd count without --isolated-etcd, but should")
	}
}

