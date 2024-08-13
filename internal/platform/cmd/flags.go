package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// AddPersistentStringFlag adds a string flag to the command
func AddPersistentStringFlag(c *cobra.Command, flag, value, description string, isRequired bool) {
	req := ""
	if isRequired {
		req = " (required)"
	}

	c.PersistentFlags().String(flag, value, fmt.Sprintf("%s%s", description, req))

	if isRequired {
		_ = c.MarkPersistentFlagRequired(flag)
	}
}

// AddPersistentIntFlag adds a int flag to the command
func AddPersistentIntFlag(c *cobra.Command, flag string, value uint64, description string, isRequired bool) {
	req := ""
	if isRequired {
		req = " (required)"
	}

	c.PersistentFlags().Uint64(flag, value, fmt.Sprintf("%s%s", description, req))

	if isRequired {
		_ = c.MarkPersistentFlagRequired(flag)
	}
}

// AddPersistentStringArrayFlag adds a string slice flag to the command
func AddPersistentStringSliceFlag(c *cobra.Command, flag string, value []string, description string, isRequired bool) {
	req := ""
	if isRequired {
		req = " (required)"
	}

	c.PersistentFlags().StringSlice(flag, value, fmt.Sprintf("%s%s", description, req))

	if isRequired {
		_ = c.MarkPersistentFlagRequired(flag)
	}
}

// AddPersistentIntFlag adds a int flag to the command
func AddPersistentDurationFlag(c *cobra.Command, flag string, value time.Duration, description string, isRequired bool) {
	req := ""
	if isRequired {
		req = " (required)"
	}

	c.PersistentFlags().Duration(flag, value, fmt.Sprintf("%s%s", description, req))

	if isRequired {
		_ = c.MarkPersistentFlagRequired(flag)
	}
}
