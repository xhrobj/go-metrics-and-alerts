package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption/testkeys"
)

func TestLoadPrivateKey(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		privateKey, err := loadPrivateKey("")

		require.NoError(t, err)
		require.Nil(t, privateKey)
	})

	t.Run("valid private key", func(t *testing.T) {
		keys := testkeys.Generate(t)

		privateKey, err := loadPrivateKey(keys.PrivateKeyPath)

		require.NoError(t, err)
		require.NotNil(t, privateKey)
		require.True(t, keys.PrivateKey.Equal(privateKey))
	})

	t.Run("missing private key", func(t *testing.T) {
		privateKey, err := loadPrivateKey("missing-private-key.pem")

		require.ErrorContains(t, err, "load private key")
		require.Nil(t, privateKey)
	})
}

func TestParseTrustedSubnet(t *testing.T) {
	t.Run("empty subnet", func(t *testing.T) {
		subnet, err := parseTrustedSubnet("")

		require.NoError(t, err)
		require.Nil(t, subnet)
	})

	t.Run("valid subnet", func(t *testing.T) {
		subnet, err := parseTrustedSubnet("192.168.1.0/24")

		require.NoError(t, err)
		require.Equal(t, "192.168.1.0/24", subnet.String())
	})

	t.Run("invalid subnet", func(t *testing.T) {
		subnet, err := parseTrustedSubnet("invalid-cidr")

		require.ErrorContains(t, err, `parse trusted subnet "invalid-cidr"`)
		require.Nil(t, subnet)
	})
}
