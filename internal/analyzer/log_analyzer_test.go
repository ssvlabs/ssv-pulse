package analyzer

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/ssvlabsinfra/ssv-pulse/configs"
)

func TestProposeAnalyze(t *testing.T) {
	const testLogFilePath = "./test/propose_test.log"
	rootCMD := &cobra.Command{
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := viper.Unmarshal(&configs.Values)
			require.NoError(t, err)
		},
	}
	rootCMD.AddCommand(CMD)

	t.Run("test propose logs - cluster", func(t *testing.T) {
		args := []string{command, fmt.Sprintf("--%s", logFilePathFlag), testLogFilePath, fmt.Sprintf("--%s", operatorsFlag), "54,178,225,226,227,228,229", fmt.Sprintf("--%s", clusterFlag)}
		rootCMD.SetArgs(args)
		err := rootCMD.Execute()
		require.NoError(t, err)
		resetFlags(rootCMD)
	})
	t.Run("test propose logs - not cluster", func(t *testing.T) {
		args := []string{command, fmt.Sprintf("--%s", logFilePathFlag), testLogFilePath, fmt.Sprintf("--%s", operatorsFlag), "54,178,225,226,227,228,229"}
		rootCMD.SetArgs(args)
		err := rootCMD.Execute()
		require.NoError(t, err)
		resetFlags(rootCMD)
	})
	t.Run("test propose logs - all", func(t *testing.T) {
		args := []string{command, fmt.Sprintf("--%s", logFilePathFlag), testLogFilePath}
		rootCMD.SetArgs(args)
		err := rootCMD.Execute()
		require.NoError(t, err)
		resetFlags(rootCMD)
	})
}

func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Value.Type() == "stringSlice" {
			value := reflect.ValueOf(flag.Value).Elem().FieldByName("value")
			ptr := (*[]string)(unsafe.Pointer(value.Pointer()))
			*ptr = make([]string, 0)
		}
	})
	for _, cmd := range cmd.Commands() {
		resetFlags(cmd)
	}
}
