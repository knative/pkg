package helpers

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyz"
	randSuffixLen = 8
	sep           = "-"
)

var (
	r        *rand.Rand
	rndMutex sync.Mutex
)

func init() {
	r = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
}

// AppendRandomString will generate a random string that begins with prefix.
// This is useful if you want to make sure that your tests can run at the same
// time against the same environment without conflicting.
// This method will use "-" as the separator between the prefix and
// the random suffix.
// This method will seed rand with the current time when the package is initialized.
func AppendRandomString(prefix string) string {
	suffix := make([]byte, randSuffixLen)

	rndMutex.Lock()
	defer rndMutex.Unlock()

	for i := range suffix {
		suffix[i] = letterBytes[r.Intn(len(letterBytes))]
	}

	return strings.Join([]string{prefix, string(suffix)}, sep)
}
