package updater

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunScheduledCheckAutoUpdateAppliesAndClearsPending(t *testing.T) {
	var applied int
	var emitted []*UpdateInfo

	service := &Service{
		autoUpdate: func() bool { return true },
		onPending: func(info *UpdateInfo) {
			emitted = append(emitted, cloneUpdateInfo(info))
		},
		applyUpdateFn: func(context.Context) error {
			applied++
			return nil
		},
		pendingUpdate: &UpdateInfo{Version: "v1.2.3", ReleaseNotes: "pending"},
	}

	err := service.runScheduledCheck(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, applied)
	require.Nil(t, service.GetPendingUpdate())
	require.Len(t, emitted, 1)
	require.Nil(t, emitted[0])
}

func TestRunScheduledCheckManualModeCachesPendingUpdate(t *testing.T) {
	var applied int
	var emitted []*UpdateInfo
	expected := &UpdateInfo{Version: "v1.2.3", ReleaseNotes: "notes"}

	service := &Service{
		autoUpdate: func() bool { return false },
		onPending: func(info *UpdateInfo) {
			emitted = append(emitted, cloneUpdateInfo(info))
		},
		checkForUpdateFn: func(context.Context) (*UpdateInfo, error) {
			return expected, nil
		},
		applyUpdateFn: func(context.Context) error {
			applied++
			return nil
		},
	}

	err := service.runScheduledCheck(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, applied)
	require.Equal(t, expected, service.GetPendingUpdate())
	require.Equal(t, []*UpdateInfo{expected}, emitted)
}

func TestRefreshPendingUpdateDeduplicatesEvents(t *testing.T) {
	var emitted int
	expected := &UpdateInfo{Version: "v1.2.3", ReleaseNotes: "notes"}

	service := &Service{
		onPending: func(*UpdateInfo) {
			emitted++
		},
		checkForUpdateFn: func(context.Context) (*UpdateInfo, error) {
			return expected, nil
		},
	}

	_, err := service.RefreshPendingUpdate(context.Background())
	require.NoError(t, err)
	_, err = service.RefreshPendingUpdate(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, emitted)
	require.Equal(t, expected, service.GetPendingUpdate())
}

func TestRefreshPendingUpdateClearsPendingState(t *testing.T) {
	var emitted []*UpdateInfo

	service := &Service{
		onPending: func(info *UpdateInfo) {
			emitted = append(emitted, cloneUpdateInfo(info))
		},
		checkForUpdateFn: func(context.Context) (*UpdateInfo, error) {
			return nil, nil
		},
		pendingUpdate: &UpdateInfo{Version: "v1.2.3", ReleaseNotes: "notes"},
	}

	info, err := service.RefreshPendingUpdate(context.Background())
	require.NoError(t, err)
	require.Nil(t, info)
	require.Nil(t, service.GetPendingUpdate())
	require.Len(t, emitted, 1)
	require.Nil(t, emitted[0])
}
