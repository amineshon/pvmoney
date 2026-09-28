package api

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"pvmoney/internal/store"
)

const maxLogoBytes = 2 << 20

func (s *Server) uploadDebtLogo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	r.Body = http.MaxBytesReader(w, r.Body, maxLogoBytes+64*1024)
	if err := r.ParseMultipartForm(maxLogoBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid: logo too large"})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid: logo required"})
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxLogoBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid: logo unreadable"})
		return
	}
	if len(raw) == 0 || len(raw) > maxLogoBytes {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid: logo too large"})
		return
	}
	mime, err := sniffImage(raw)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid: unsupported image"})
		return
	}
	if err := s.store.SetDebtLogo(r.Context(), id, mime, raw); err != nil {
		writeErr(w, err)
		return
	}
	item, err := s.store.GetDebt(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) getDebtLogo(w http.ResponseWriter, r *http.Request) {
	mime, data, err := s.store.DebtLogo(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) deleteDebtLogo(w http.ResponseWriter, r *http.Request) {
	if err := s.store.ClearDebtLogo(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func sniffImage(b []byte) (string, error) {
	if len(b) < 12 {
		return "", store.ErrInvalid
	}
	switch {
	case bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg", nil
	case bytes.HasPrefix(b, []byte{0x89, 0x50, 0x4E, 0x47}):
		return "image/png", nil
	case bytes.HasPrefix(b, []byte("GIF8")):
		return "image/gif", nil
	case bytes.HasPrefix(b[8:], []byte("WEBP")) && strings.HasPrefix(string(b[:4]), "RIFF"):
		return "image/webp", nil
	default:
		return "", store.ErrInvalid
	}
}
