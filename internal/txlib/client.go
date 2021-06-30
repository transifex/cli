package txlib

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
)

func GetClient(cacert string) (http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if cacert != "" {
		file, err := os.Open(cacert)
		if err != nil {
			return http.Client{}, err
		}
		data, err := io.ReadAll(file)
		if err != nil {
			return http.Client{}, err
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(data) {
			return http.Client{}, fmt.Errorf(
				"could not load certificates from file '%s'",
				cacert,
			)
		}

		transport.TLSClientConfig = &tls.Config{RootCAs: certPool}
	}

	return http.Client{Transport: transport}, nil
}
