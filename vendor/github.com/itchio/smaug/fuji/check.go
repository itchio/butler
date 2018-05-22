//+build windows

package fuji

import (
	"syscall"

	"github.com/itchio/ox/syscallex"
	"github.com/itchio/ox/winox"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

func (i *instance) Check(consumer *state.Consumer) error {
	consumer.Opf("Retrieving player data from registry...")
	creds, err := i.GetCredentials()
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Statf("Sandbox user is (%s)", creds.Username)
	consumer.Statf("Sandbox password is (%s)", creds.Password)

	consumer.Opf("Trying to log in...")

	token, err := winox.Logon(creds.Username, ".", creds.Password)

	if err != nil {
		rescued := false

		if en, ok := winox.AsErrno(err); ok {
			switch en {
			case syscallex.ERROR_PASSWORD_EXPIRED,
				syscallex.ERROR_PASSWORD_MUST_CHANGE:
				// Some Windows versions (10 for example) expire password automatically.
				// Thankfully, we can renew it without administrator access, simply by using the old one.
				consumer.Opf("Password has expired, setting new password...")
				newPassword := generatePassword()

				err := syscallex.NetUserChangePassword(
					nil, // domainname
					syscall.StringToUTF16Ptr(creds.Username),
					syscall.StringToUTF16Ptr(creds.Password),
					syscall.StringToUTF16Ptr(newPassword),
				)
				if err != nil {
					return errors.WithStack(err)
				}

				creds.Password = newPassword
				err = i.saveCredentials(creds)
				if err != nil {
					return errors.WithStack(err)
				}

				token, err = winox.Logon(creds.Username, ".", creds.Password)
				if err != nil {
					return errors.WithStack(err)
				}

				consumer.Statf("Set new password successfully!")

				rescued = true
			}
		}

		if !rescued {
			return errors.WithStack(err)
		}
	}
	defer winox.SafeRelease(uintptr(token))

	consumer.Statf("Everything looks good!")

	return nil
}
