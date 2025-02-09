package planar

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnplanar(t *testing.T) {
	b := []byte{1, 2, 3, 4, 5, 6, 7, 8, 11, 12, 13, 14, 15, 16, 17, 18}
	orig := bytes.NewReader(b)
	unplanared := NewUnplanarReader(orig, 2, 2, 8)

	r := make([]byte, 8)
	n, err := unplanared.Read(r)
	require.NoError(t, err)
	require.Equal(t, 8, n)
	require.Equal(t, []byte{1, 2, 5, 6, 3, 4, 7, 8}, r)

	n, err = unplanared.Read(r)
	require.NoError(t, err)
	require.Equal(t, 8, n)
	require.Equal(t, []byte{11, 12, 15, 16, 13, 14, 17, 18}, r)
}
