package webhook

import (
	"context"
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

type Webhook struct {
	server   *http.Server
	listener net.Listener
	handler  *handler
}

func NewWebhook(listener net.Listener) *Webhook {
	return &Webhook{
		server:   &http.Server{},
		handler:  &handler{},
		listener: listener,
	}
}

func (f *Webhook) Name() string {
	return "webhook"
}

func (f *Webhook) Serve(_ context.Context) error {
	r := mux.NewRouter()

	r.HandleFunc("/coinbase-webhook", f.handler.ReceiveCoinbaseWebhook).Methods(http.MethodPost)

	f.server.Handler = r
	f.server.Serve(f.listener)
	return nil
}

func (f *Webhook) Shutdown(ctx context.Context) error {
	f.server.Shutdown(ctx)
	return nil
}
