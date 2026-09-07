package security

import (
	"crypto/tls"
	"errors"
	"fmt"
)

var (
	ErrTLSVersionTooLow  = errors.New("TLS version below 1.2 — connection rejected")
	ErrTLSWeakCipher     = errors.New("weak cipher suite detected — connection rejected")
)

var weakCipherSuites = map[uint16]bool{
	tls.TLS_RSA_WITH_RC4_128_SHA:                true,
	tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:           true,
	tls.TLS_RSA_WITH_AES_128_CBC_SHA:            true,
	tls.TLS_RSA_WITH_AES_256_CBC_SHA:            true,

	tls.TLS_RSA_WITH_AES_128_GCM_SHA256:         true,
	tls.TLS_RSA_WITH_AES_256_GCM_SHA384:         true,
	tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:          true,
	tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:     true,
}

type TLSGateway struct {
	minVersion    uint16
	allowedCerts  map[string]bool
}

func NewTLSGateway() *TLSGateway {
	return &TLSGateway{
		minVersion:   tls.VersionTLS12,
		allowedCerts: make(map[string]bool),
	}
}

func (g *TLSGateway) CheckTLSVersion(version uint16) error {
	if version < g.minVersion {
		return fmt.Errorf("%w: got 0x%04x, require >= 0x%04x", ErrTLSVersionTooLow, version, g.minVersion)
	}
	return nil
}

func (g *TLSGateway) CheckCipherSuite(cipher uint16) error {
	if weakCipherSuites[cipher] {
		return fmt.Errorf("%w: cipher 0x%04x", ErrTLSWeakCipher, cipher)
	}
	return nil
}

func (g *TLSGateway) TLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}
}

func (g *TLSGateway) ValidateConnection(state tls.ConnectionState) error {
	if err := g.CheckTLSVersion(state.Version); err != nil {
		return err
	}
	if err := g.CheckCipherSuite(state.CipherSuite); err != nil {
		return err
	}
	return nil
}