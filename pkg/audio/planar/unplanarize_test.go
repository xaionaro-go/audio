package planar

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func clean(s string) string {
	return strings.ReplaceAll(s, " ", "")
}

func TestUnplanarize(t *testing.T) {
	b := must(hex.DecodeString(clean("00010203 04050607 08090A0B 0C0D0E0F 10111213 14151617 18191A1B 1C1D1E1F")))
	r := make([]byte, len(b))
	err := Unplanarize(2, 4, r, b)
	require.NoError(t, err)
	require.Equal(t, must(hex.DecodeString(clean("00010203 10111213 04050607 14151617 08090A0B 18191A1B 0C0D0E0F 1C1D1E1F"))), r, spew.Sdump(b))
}

func TestUnplanarizeReader(t *testing.T) {
	b := must(hex.DecodeString(clean("00010203 04050607 08090A0B 0C0D0E0F 10111213 14151617 18191A1B 1C1D1E1F")))
	orig := bytes.NewReader(b)
	unplanared := NewUnplanarizeReader(orig, 2, 4, 65536)

	r := make([]byte, len(b))
	n, err := unplanared.Read(r)
	require.NoError(t, err)
	require.Equal(t, len(b), n)
	require.Equal(t, must(hex.DecodeString(clean("00010203 10111213 04050607 14151617 08090A0B 18191A1B 0C0D0E0F 1C1D1E1F"))), r, spew.Sdump(b))
}
