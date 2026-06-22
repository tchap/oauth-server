package oauthserver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func generateCA(t *testing.T) (certPEM []byte, keyPEM []byte, cert *x509.Certificate, key *ecdsa.PrivateKey) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatal(err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, cert, key
}

func generateServerCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) tls.Certificate {
	t.Helper()

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  serverKey,
	}
}

func startTLSServer(t *testing.T, serverCert tls.Certificate) *httptest.Server {
	t.Helper()

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

// TestDynamicCARoundTripper_ProxyCAOnly verifies that a transport with only
// a proxy CA (no IdP CA) can connect to a server signed by that CA.
func TestDynamicCARoundTripper_ProxyCAOnly(t *testing.T) {
	proxyCAPEM, _, proxyCACert, proxyCAKey := generateCA(t)
	serverCert := generateServerCert(t, proxyCACert, proxyCAKey)
	server := startTLSServer(t, serverCert)

	proxyCAFile := filepath.Join(t.TempDir(), "proxy-ca.pem")
	writeFile(t, proxyCAFile, proxyCAPEM)

	rt, err := newDynamicCARoundTripper(proxyCAFile, "", "", "")
	if err != nil {
		t.Fatalf("newDynamicCARoundTripper: %v", err)
	}

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestDynamicCARoundTripper_CombinedCAs verifies that a transport with both
// IdP CA and proxy CA can connect to servers signed by either CA.
func TestDynamicCARoundTripper_CombinedCAs(t *testing.T) {
	// IdP CA and its server
	idpCAPEM, _, idpCACert, idpCAKey := generateCA(t)
	idpServerCert := generateServerCert(t, idpCACert, idpCAKey)
	idpServer := startTLSServer(t, idpServerCert)

	// Proxy CA and its server
	proxyCAPEM, _, proxyCACert, proxyCAKey := generateCA(t)
	proxyServerCert := generateServerCert(t, proxyCACert, proxyCAKey)
	proxyServer := startTLSServer(t, proxyServerCert)

	dir := t.TempDir()
	idpCAFile := filepath.Join(dir, "idp-ca.pem")
	proxyCAFile := filepath.Join(dir, "proxy-ca.pem")
	writeFile(t, idpCAFile, idpCAPEM)
	writeFile(t, proxyCAFile, proxyCAPEM)

	rt, err := newDynamicCARoundTripper(proxyCAFile, idpCAFile, "", "")
	if err != nil {
		t.Fatalf("newDynamicCARoundTripper: %v", err)
	}

	// Should connect to IdP server (IdP CA)
	req1, _ := http.NewRequest("GET", idpServer.URL, nil)
	resp, err := rt.RoundTrip(req1)
	if err != nil {
		t.Fatalf("RoundTrip to IdP server failed: %v", err)
	}
	resp.Body.Close()

	// Should connect to proxy server (proxy CA)
	req2, _ := http.NewRequest("GET", proxyServer.URL, nil)
	resp, err = rt.RoundTrip(req2)
	if err != nil {
		t.Fatalf("RoundTrip to proxy server failed: %v", err)
	}
	resp.Body.Close()
}

// TestDynamicCARoundTripper_ProxyCAReload verifies that when the proxy CA file
// changes on disk, the transport picks up the new CA after RunOnce triggers
// the reload.
func TestDynamicCARoundTripper_ProxyCAReload(t *testing.T) {
	proxyCA1PEM, _, proxyCA1Cert, proxyCA1Key := generateCA(t)
	server1Cert := generateServerCert(t, proxyCA1Cert, proxyCA1Key)
	server1 := startTLSServer(t, server1Cert)

	proxyCA2PEM, _, proxyCA2Cert, proxyCA2Key := generateCA(t)
	server2Cert := generateServerCert(t, proxyCA2Cert, proxyCA2Key)
	server2 := startTLSServer(t, server2Cert)

	proxyCAFile := filepath.Join(t.TempDir(), "proxy-ca.pem")
	writeFile(t, proxyCAFile, proxyCA1PEM)

	rt, err := newDynamicCARoundTripper(proxyCAFile, "", "", "")
	if err != nil {
		t.Fatalf("newDynamicCARoundTripper: %v", err)
	}

	// server1 should work with proxyCA1
	req1, _ := http.NewRequest("GET", server1.URL, nil)
	resp, err := rt.RoundTrip(req1)
	if err != nil {
		t.Fatalf("RoundTrip to server1 failed: %v", err)
	}
	resp.Body.Close()

	// server2 should fail with proxyCA1
	req2, _ := http.NewRequest("GET", server2.URL, nil)
	_, err = rt.RoundTrip(req2)
	if err == nil {
		t.Fatal("expected RoundTrip to server2 to fail with proxyCA1")
	}

	// Rotate proxy CA to CA2, trigger reload
	writeFile(t, proxyCAFile, proxyCA2PEM)
	if err := rt.proxyCAContent.RunOnce(t.Context()); err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}

	// server2 should now work with proxyCA2
	req3, _ := http.NewRequest("GET", server2.URL, nil)
	resp, err = rt.RoundTrip(req3)
	if err != nil {
		t.Fatalf("RoundTrip to server2 failed after proxy CA reload: %v", err)
	}
	resp.Body.Close()
}

// TestDynamicCARoundTripper_ErrorResilience verifies that corrupting the proxy
// CA file does not break existing connections — the old transport is preserved.
func TestDynamicCARoundTripper_ErrorResilience(t *testing.T) {
	proxyCAPEM, _, proxyCACert, proxyCAKey := generateCA(t)
	serverCert := generateServerCert(t, proxyCACert, proxyCAKey)
	server := startTLSServer(t, serverCert)

	proxyCAFile := filepath.Join(t.TempDir(), "proxy-ca.pem")
	writeFile(t, proxyCAFile, proxyCAPEM)

	rt, err := newDynamicCARoundTripper(proxyCAFile, "", "", "")
	if err != nil {
		t.Fatalf("newDynamicCARoundTripper: %v", err)
	}

	// Corrupt the proxy CA file
	writeFile(t, proxyCAFile, []byte("not-a-certificate"))
	if err := rt.proxyCAContent.RunOnce(t.Context()); err == nil {
		t.Fatal("expected RunOnce to fail with corrupt CA")
	}

	// Old transport should still work
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip should still work after failed reload: %v", err)
	}
	resp.Body.Close()
}

// TestDynamicCARoundTripper_InvalidProxyCAFile verifies that constructing
// a dynamicCARoundTripper with a nonexistent proxy CA file fails.
func TestDynamicCARoundTripper_InvalidProxyCAFile(t *testing.T) {
	_, err := newDynamicCARoundTripper("/nonexistent/proxy-ca.pem", "", "", "")
	if err == nil {
		t.Fatal("expected error for nonexistent proxy CA file")
	}
}
