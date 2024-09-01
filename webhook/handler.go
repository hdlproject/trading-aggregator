package webhook

import (
	"fmt"
	"io"
	"net/http"
)

type handler struct {
}

func (h handler) ReceiveCoinbaseWebhook(w http.ResponseWriter, r *http.Request) {
	resBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println(string(resBody))
}
