package proxy

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"sync"

	"zhiyuwaf/internal/acme"
)

type Listener struct {
	addr        string
	tlsAddr     string
	handler     *Handler
	tlsCertFile string
	tlsKeyFile  string
	acmeManager *acme.Manager
	server      *http.Server
	tlsServer   *http.Server
}

func NewListener(addr, tlsAddr string, handler *Handler, tlsCertFile, tlsKeyFile string) *Listener {
	return &Listener{
		addr:        addr,
		tlsAddr:     tlsAddr,
		handler:     handler,
		tlsCertFile: tlsCertFile,
		tlsKeyFile:  tlsKeyFile,
	}
}

func (l *Listener) SetACMEManager(m *acme.Manager) {
	l.acmeManager = m
}

func (l *Listener) Start(ctx context.Context) error {
	cfg := l.handler.getConfig()

	// Always start HTTP server
	l.server = &http.Server{
		Addr:         l.addr,
		Handler:      l.handler,
		ReadTimeout:  cfg.readTimeout,
		WriteTimeout: cfg.writeTimeout,
	}

	// If ACME is enabled, wrap HTTP handler for ACME challenges
	if l.acmeManager != nil && l.acmeManager.Enabled() {
		l.server.Handler = l.acmeManager.HTTPHandler(l.handler)
		log.Println("ACME HTTP-01 challenge handler enabled on HTTP server")
	}

	log.Printf("proxy listening on %s", l.addr)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// Start HTTP
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := l.server.ListenAndServe()
		if err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Start HTTPS if TLS is configured
	if l.tlsCertFile != "" && l.tlsKeyFile != "" && l.tlsAddr != "" {
		l.tlsServer = &http.Server{
			Addr:         l.tlsAddr,
			Handler:      l.handler,
			ReadTimeout:  cfg.readTimeout,
			WriteTimeout: cfg.writeTimeout,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		log.Printf("proxy listening on %s (TLS + HTTP/2)", l.tlsAddr)

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := l.tlsServer.ListenAndServeTLS(l.tlsCertFile, l.tlsKeyFile)
			if err != http.ErrServerClosed {
				errCh <- err
			}
		}()
	} else if l.acmeManager != nil && l.acmeManager.Enabled() && l.tlsAddr != "" {
		// ACME mode: use autocert for TLS
		l.tlsServer = &http.Server{
			Addr:         l.tlsAddr,
			Handler:      l.handler,
			ReadTimeout:  cfg.readTimeout,
			WriteTimeout: cfg.writeTimeout,
			TLSConfig:    l.acmeManager.TLSConfig(),
		}
		log.Printf("proxy listening on %s (ACME auto-TLS)", l.tlsAddr)

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := l.tlsServer.ListenAndServeTLS("", "")
			if err != http.ErrServerClosed {
				errCh <- err
			}
		}()
	}

	// Shutdown on context cancel
	go func() {
		<-ctx.Done()
		l.server.Shutdown(context.Background())
		if l.tlsServer != nil {
			l.tlsServer.Shutdown(context.Background())
		}
	}()

	// Wait for first error or all done
	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}
