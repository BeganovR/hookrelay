package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

func ReceiverHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1 MB is limit for 1 webhook
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("reading body is failed", "error:", err)
		http.Error(w, fmt.Sprintf("Error: reading body. Desc: %v\n", err), http.StatusBadRequest)
		return
	}
	hookId := r.PathValue("id")

	slog.Info("ID and Body are received", "ID", hookId, "Body", string(body))
	w.WriteHeader(http.StatusAccepted)
	if _, err = w.Write([]byte("Webhook received\n")); err != nil {
		slog.Error("writing response failed", "error", err)
	}
}
