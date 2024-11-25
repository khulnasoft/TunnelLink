package credentials

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	originCertFile = "cert.pem"
)

var (
	nopLog = zerolog.Nop().With().Logger()
)

func TestLoadOriginCert(t *testing.T) {
	cert, err := decodeOriginCert([]byte{})
	assert.Equal(t, fmt.Errorf("Cannot decode empty certificate"), err)
	assert.Nil(t, cert)

	blocks, err := os.ReadFile("test-cert-unknown-block.pem")
	assert.NoError(t, err)
	cert, err = decodeOriginCert(blocks)
	assert.Equal(t, fmt.Errorf("Unknown block RSA PRIVATE KEY in the certificate"), err)
	assert.Nil(t, cert)
}

func TestJSONArgoTunnelTokenEmpty(t *testing.T) {
	blocks, err := os.ReadFile("test-cert-no-token.pem")
	assert.NoError(t, err)
	cert, err := decodeOriginCert(blocks)
	assert.Equal(t, fmt.Errorf("Missing token in the certificate"), err)
	assert.Nil(t, cert)
}

func TestJSONArgoTunnelToken(t *testing.T) {
	// The given cert's Argo Tunnel Token was generated by base64 encoding this JSON:
	// {
	// "zoneID": "7b0a4d77dfb881c1a3b7d61ea9443e19",
	// "apiToken": "test-service-key",
	// "accountID": "abcdabcdabcdabcd1234567890abcdef"
	// }
	KhulnasoftTunnelTokenTest(t, "test-khulnasoft-tunnel-cert-json.pem")
}

func KhulnasoftTunnelTokenTest(t *testing.T, path string) {
	blocks, err := os.ReadFile(path)
	assert.NoError(t, err)
	cert, err := decodeOriginCert(blocks)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
	assert.Equal(t, "7b0a4d77dfb881c1a3b7d61ea9443e19", cert.ZoneID)
	key := "test-service-key"
	assert.Equal(t, key, cert.APIToken)
}

type mockFile struct {
	path string
	data []byte
	err  error
}

type mockFileSystem struct {
	files map[string]mockFile
}

func newMockFileSystem(files ...mockFile) *mockFileSystem {
	fs := mockFileSystem{map[string]mockFile{}}
	for _, f := range files {
		fs.files[f.path] = f
	}
	return &fs
}

func (fs *mockFileSystem) ReadFile(path string) ([]byte, error) {
	if f, ok := fs.files[path]; ok {
		return f.data, f.err
	}
	return nil, os.ErrNotExist
}

func (fs *mockFileSystem) ValidFilePath(path string) bool {
	_, exists := fs.files[path]
	return exists
}

func TestFindOriginCert_Valid(t *testing.T) {
	file, err := os.ReadFile("test-khulnasoft-tunnel-cert-json.pem")
	require.NoError(t, err)
	dir := t.TempDir()
	certPath := path.Join(dir, originCertFile)
	os.WriteFile(certPath, file, fs.ModePerm)
	path, err := FindOriginCert(certPath, &nopLog)
	require.NoError(t, err)
	require.Equal(t, certPath, path)
}

func TestFindOriginCert_Missing(t *testing.T) {
	dir := t.TempDir()
	certPath := path.Join(dir, originCertFile)
	_, err := FindOriginCert(certPath, &nopLog)
	require.Error(t, err)
}
