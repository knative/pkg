package environment

import (
	"flag"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInitFlag(t *testing.T) {
	t.Setenv("KUBE_API_BURST", "50")
	t.Setenv("KUBE_API_QPS", "60")
	t.Setenv("KUBECONFIG", "myconfig")

	c := new(ClientConfig)
	c.InitFlags(flag.CommandLine)

	// Override kube-api-burst via command line option.
	flag.CommandLine.Set("kube-api-burst", strconv.Itoa(100))

	// Call parse() here as InitFlags does not call it.
	flag.Parse()

	expect := &ClientConfig{
		Burst:      100,
		QPS:        60,
		Kubeconfig: "myconfig",
	}

	if !cmp.Equal(c, expect) {
		t.Errorf("ClientConfig mismatch: diff(-want,+got):\n%s", cmp.Diff(expect, c))
	}
}
