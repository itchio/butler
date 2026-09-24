package integrate

import (
	"testing"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
)

const deviceClientID = "device-client"

type deviceLoginResult struct {
	res *butlerd.ProfileLoginWithDeviceResult
	err error
}

func startDeviceLogin(rc *butlerd.RequestContext, h *handler, id string) (chan butlerd.ProfileLoginWithDeviceChallengeNotification, chan deviceLoginResult) {
	challenges := make(chan butlerd.ProfileLoginWithDeviceChallengeNotification, 4)
	messages.ProfileLoginWithDeviceChallenge.Register(h, func(n butlerd.ProfileLoginWithDeviceChallengeNotification) {
		challenges <- n
	})

	done := make(chan deviceLoginResult, 1)
	go func() {
		res, err := messages.ProfileLoginWithDevice.TestCall(rc, butlerd.ProfileLoginWithDeviceParams{
			ID:       id,
			ClientID: deviceClientID,
		})
		done <- deviceLoginResult{res, err}
	}()
	return challenges, done
}

func waitChallenge(t *testing.T, challenges chan butlerd.ProfileLoginWithDeviceChallengeNotification) butlerd.ProfileLoginWithDeviceChallengeNotification {
	select {
	case n := <-challenges:
		return n
	case <-time.After(10 * time.Second):
		t.Fatal("no device sign-in challenge")
		return butlerd.ProfileLoginWithDeviceChallengeNotification{}
	}
}

func waitDone(t *testing.T, done chan deviceLoginResult) deviceLoginResult {
	select {
	case r := <-done:
		return r
	case <-time.After(10 * time.Second):
		t.Fatal("device sign-in did not finish")
		return deviceLoginResult{}
	}
}

func findDeviceAuth(t *testing.T, store *mitch.Store, n butlerd.ProfileLoginWithDeviceChallengeNotification) *mitch.DeviceAuth {
	da := store.FindDeviceAuthByUserCode(n.UserCode)
	if da == nil {
		t.Fatalf("no device auth for user code %q", n.UserCode)
	}
	return da
}

func Test_ProfileLoginWithDevice(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	store := bi.Server.Store()
	user := store.MakeUser("Phone Approver")

	messages.ProfileLoginWithDeviceRequestDeviceInfo.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.ProfileLoginWithDeviceRequestDeviceInfoParams) (*butlerd.ProfileLoginWithDeviceRequestDeviceInfoResult, error) {
		assert.Equal("first", params.ID)
		return &butlerd.ProfileLoginWithDeviceRequestDeviceInfoResult{DeviceInfo: `{"platform":"test"}`}, nil
	})

	challenges, done := startDeviceLogin(rc, h, "first")
	n := waitChallenge(t, challenges)
	assert.Equal("first", n.ID)
	assert.Contains(n.URL, "?code="+n.UserCode)
	assert.EqualValues(mitch.DeviceAuthExpiresIn, n.ExpiresIn)

	_, err := messages.ProfileLoginWithDevice.TestCall(rc, butlerd.ProfileLoginWithDeviceParams{ID: "second", ClientID: deviceClientID})
	assert.Error(err)
	assert.EqualValues(butlerd.CodeProfileLoginWithDeviceInProgress, err.(*jsonrpc2.Error).Code)

	findDeviceAuth(t, store, n).Approve(user)

	r := waitDone(t, done)
	must(r.err)
	assert.Equal("Phone Approver", r.res.Profile.User.DisplayName)

	keys := store.ListAPIKeysByUser(user.ID)
	if assert.Len(keys, 1) {
		assert.Equal(`{"platform":"test"}`, keys[0].DeviceInfo)
	}

	list, err := messages.ProfileList.TestCall(rc, butlerd.ProfileListParams{})
	must(err)
	if assert.Len(list.Profiles, 1) {
		assert.EqualValues(user.ID, list.Profiles[0].ID)
	}
}

func Test_ProfileLoginWithDevice_ExpiredThenApproved(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	store := bi.Server.Store()
	user := store.MakeUser("Slow Approver")

	// No device info handler: the exchange goes ahead without any
	challenges, done := startDeviceLogin(rc, h, "expiring")
	first := waitChallenge(t, challenges)
	findDeviceAuth(t, store, first).Expire()

	second := waitChallenge(t, challenges)
	assert.NotEqual(first.UserCode, second.UserCode)
	findDeviceAuth(t, store, second).Approve(user)

	r := waitDone(t, done)
	must(r.err)
	assert.EqualValues(user.ID, r.res.Profile.ID)

	keys := store.ListAPIKeysByUser(user.ID)
	if assert.Len(keys, 1) {
		assert.Equal("", keys[0].DeviceInfo)
	}
}

func Test_ProfileLoginWithDevice_Denied(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	store := bi.Server.Store()

	challenges, done := startDeviceLogin(rc, h, "denied")
	findDeviceAuth(t, store, waitChallenge(t, challenges)).Deny()

	r := waitDone(t, done)
	assert.Error(r.err)
	assert.EqualValues(butlerd.CodeProfileLoginWithDeviceDenied, r.err.(*jsonrpc2.Error).Code)
}

func Test_ProfileLoginWithDevice_Cancel(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	cancelRes, err := messages.ProfileLoginWithDeviceCancel.TestCall(rc, butlerd.ProfileLoginWithDeviceCancelParams{ID: "nope"})
	must(err)
	assert.False(cancelRes.DidCancel)

	challenges, done := startDeviceLogin(rc, h, "cancelled")
	waitChallenge(t, challenges)

	cancelRes, err = messages.ProfileLoginWithDeviceCancel.TestCall(rc, butlerd.ProfileLoginWithDeviceCancelParams{ID: "cancelled"})
	must(err)
	assert.True(cancelRes.DidCancel)

	r := waitDone(t, done)
	assert.Error(r.err)
	assert.EqualValues(butlerd.CodeOperationCancelled, r.err.(*jsonrpc2.Error).Code)
}

func Test_ProfileLoginWithDevice_CancelWhileAskingDeviceInfo(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	store := bi.Server.Store()
	user := store.MakeUser("Silent Client")

	asked := make(chan struct{}, 1)
	release := make(chan struct{})
	messages.ProfileLoginWithDeviceRequestDeviceInfo.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.ProfileLoginWithDeviceRequestDeviceInfoParams) (*butlerd.ProfileLoginWithDeviceRequestDeviceInfoResult, error) {
		asked <- struct{}{}
		<-release
		return &butlerd.ProfileLoginWithDeviceRequestDeviceInfoResult{}, nil
	})
	defer close(release)

	challenges, done := startDeviceLogin(rc, h, "stuck")
	findDeviceAuth(t, store, waitChallenge(t, challenges)).Approve(user)

	select {
	case <-asked:
	case <-time.After(10 * time.Second):
		t.Fatal("device info was never requested")
	}

	cancelRes, err := messages.ProfileLoginWithDeviceCancel.TestCall(rc, butlerd.ProfileLoginWithDeviceCancelParams{ID: "stuck"})
	must(err)
	assert.True(cancelRes.DidCancel)

	r := waitDone(t, done)
	assert.Error(r.err)
	assert.EqualValues(butlerd.CodeOperationCancelled, r.err.(*jsonrpc2.Error).Code)

	// The unanswered request must not block the next sign-in
	challenges, done = startDeviceLogin(rc, h, "next")
	waitChallenge(t, challenges)
	_, err = messages.ProfileLoginWithDeviceCancel.TestCall(rc, butlerd.ProfileLoginWithDeviceCancelParams{ID: "next"})
	must(err)
	waitDone(t, done)
}
