//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseBackgroundStartDelay(t *testing.T) {
	require.Zero(t, parseBackgroundStartDelay(""))
	require.Zero(t, parseBackgroundStartDelay("0"))
	require.Equal(t, 5*time.Minute, parseBackgroundStartDelay("300"))
	require.Zero(t, parseBackgroundStartDelay("-1"))
	require.Zero(t, parseBackgroundStartDelay("3601"))
	require.Zero(t, parseBackgroundStartDelay("invalid"))
}
