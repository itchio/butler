package profile

import (
	"context"
	"sync"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

// One sign-in at a time, so the client never has two codes on screen.
var deviceLoginMu sync.Mutex

func LoginWithDevice(rc *butlerd.RequestContext, params butlerd.ProfileLoginWithDeviceParams) (*butlerd.ProfileLoginWithDeviceResult, error) {
	if !deviceLoginMu.TryLock() {
		return nil, butlerd.CodeProfileLoginWithDeviceInProgress
	}
	defer deviceLoginMu.Unlock()

	ctx, cleanup := rc.MakeCancelable(params.ID)
	defer cleanup()

	consumer := rc.Consumer
	client := rc.RootClient()

	for {
		verifier, challenge, err := itchio.GeneratePKCE()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		start, err := client.DeviceAuth(ctx, itchio.DeviceAuthParams{
			ClientID:      params.ClientID,
			CodeChallenge: challenge,
		})
		if err != nil {
			return nil, unlessCancelled(ctx, err)
		}

		err = messages.ProfileLoginWithDeviceChallenge.Notify(rc, butlerd.ProfileLoginWithDeviceChallengeNotification{
			ID:        params.ID,
			URL:       start.VerificationURIComplete,
			UserCode:  start.UserCode,
			ExpiresIn: start.ExpiresIn,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		deadline := time.Now().Add(time.Duration(start.ExpiresIn) * time.Second)
		interval := pollInterval(start.Interval)

		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return nil, butlerd.CodeOperationCancelled
			case <-time.After(interval):
			}

			poll, err := client.PollDeviceAuth(ctx, itchio.PollDeviceAuthParams{
				ClientID:   params.ClientID,
				DeviceCode: start.DeviceCode,
			})
			if err != nil {
				if ctx.Err() != nil {
					return nil, butlerd.CodeOperationCancelled
				}
				if itchio.IsAPIError(err) {
					return nil, errors.WithStack(err)
				}
				// Network trouble; the code stays valid until the deadline
				consumer.Warnf("Device sign-in poll: %+v", err)
				continue
			}

			switch poll.Status {
			case itchio.DeviceAuthPending:
				interval = pollInterval(poll.Interval)
			case itchio.DeviceAuthSlowDown:
				interval *= 2
			case itchio.DeviceAuthDenied:
				return nil, butlerd.CodeProfileLoginWithDeviceDenied
			case itchio.DeviceAuthExpired:
				deadline = time.Time{}
			case itchio.DeviceAuthApproved:
				return exchangeDeviceCode(ctx, rc, params, poll.Code, verifier)
			default:
				return nil, errors.Errorf("unknown device sign-in status %q", poll.Status)
			}
		}
	}
}

func pollInterval(seconds int64) time.Duration {
	if seconds < 1 {
		seconds = 1
	}
	return time.Duration(seconds) * time.Second
}

func unlessCancelled(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return butlerd.CodeOperationCancelled
	}
	return errors.WithStack(err)
}

func exchangeDeviceCode(ctx context.Context, rc *butlerd.RequestContext, params butlerd.ProfileLoginWithDeviceParams, code string, verifier string) (*butlerd.ProfileLoginWithDeviceResult, error) {
	deviceInfo, err := requestDeviceInfo(ctx, rc, params.ID)
	if err != nil {
		return nil, err
	}

	tokenRes, err := rc.RootClient().ExchangeOAuthCode(ctx, itchio.ExchangeOAuthCodeParams{
		Code:         code,
		CodeVerifier: verifier,
		RedirectURI:  itchio.DeviceRedirectURI,
		ClientID:     params.ClientID,
		DeviceInfo:   deviceInfo,
	})
	if err != nil {
		return nil, unlessCancelled(ctx, err)
	}

	profile, err := saveProfile(rc, tokenRes.Key.Key)
	if err != nil {
		return nil, unlessCancelled(ctx, err)
	}
	return &butlerd.ProfileLoginWithDeviceResult{
		Cookie:  tokenRes.Cookie,
		Profile: profile,
	}, nil
}

// A client that refuses the request shares nothing
func requestDeviceInfo(ctx context.Context, rc *butlerd.RequestContext, id string) (string, error) {
	res, err := messages.ProfileLoginWithDeviceRequestDeviceInfo.Call(rc, butlerd.ProfileLoginWithDeviceRequestDeviceInfoParams{
		ID: id,
	})
	if ctx.Err() != nil {
		return "", butlerd.CodeOperationCancelled
	}
	if err != nil {
		return "", nil
	}
	return res.DeviceInfo, nil
}

func LoginWithDeviceCancel(rc *butlerd.RequestContext, params butlerd.ProfileLoginWithDeviceCancelParams) (*butlerd.ProfileLoginWithDeviceCancelResult, error) {
	return &butlerd.ProfileLoginWithDeviceCancelResult{DidCancel: rc.CancelFuncs.Call(params.ID)}, nil
}
