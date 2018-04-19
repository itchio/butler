//+build windows

package fuji

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

/** Letters used when generating a random password */
const kLetters = "abcdefghijklmnopqrstuvwxyz"

/** Numbers used when generating a random password */
const kNumbers = "0123456789"

/** Special characters used when generating a random password */
const kSpecial = "!_?-.;+/()=&"

func randomCharFromSet(prng *rand.Rand, set string) string {
	index := prng.Intn(len(set))
	return set[index : index+1]
}

func generatePassword() string {
	pwd := ""
	prng := getPrng()

	for i := 0; i < 16; i++ {
		var token string
		switch i % 4 {
		case 0:
			token = randomCharFromSet(prng, kLetters)
		case 1:
			token = randomCharFromSet(prng, kNumbers)
		case 2:
			token = randomCharFromSet(prng, kSpecial)
		case 3:
			token = strings.ToUpper(randomCharFromSet(prng, kLetters))
		}
		pwd += token
	}
	return pwd
}

var _prng *rand.Rand
var _initPrngOnce sync.Once

func getPrng() *rand.Rand {
	_initPrngOnce.Do(func() {
		_prng = rand.New(rand.NewSource(time.Now().UnixNano()))
	})
	return _prng
}
