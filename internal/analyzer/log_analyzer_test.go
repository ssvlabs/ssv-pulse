package analyzer

import (
	"os"
	"reflect"
	"testing"
	"unsafe"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/ssvlabsinfra/ssv-pulse/internal/platform/cmd"
)

func TestProposeAnalyze(t *testing.T) {
	err := os.RemoveAll("./output/")
	require.NoError(t, err)
	version := "test.version"
	RootCmd := &cobra.Command{
		Use:   "log-analyze",
		Short: "Read and analyze ssv node logs",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
		},
	}
	RootCmd.AddCommand(CMD)
	RootCmd.AddCommand(cmd.Version)
	RootCmd.Short = "log-analyze"
	RootCmd.Version = version
	t.Run("test propose logs - cluster", func(t *testing.T) {
		args := []string{"log-analyzer", "--logFilePath", "./test/propose_test.log", "--operators", "54,178,225,226,227,228,229", "--cluster"}
		RootCmd.SetArgs(args)
		err := RootCmd.Execute()
		require.NoError(t, err)
		resetFlags(RootCmd)
	})
	t.Run("test propose logs - not cluster", func(t *testing.T) {
		args := []string{"log-analyzer", "--logFilePath", "./test/propose_test.log", "--operators", "54,178,225,226,227,228,229"}
		RootCmd.SetArgs(args)
		err := RootCmd.Execute()
		require.NoError(t, err)
		resetFlags(RootCmd)
	})
	t.Run("test propose logs - all", func(t *testing.T) {
		args := []string{"log-analyzer", "--logFilePath", "./test/propose_test.log"}
		RootCmd.SetArgs(args)
		err := RootCmd.Execute()
		require.NoError(t, err)
		resetFlags(RootCmd)
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
