package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"pvmoney/internal/models"
	"pvmoney/internal/notify"
	"pvmoney/internal/rates"
	"pvmoney/internal/store"

	"github.com/go-chi/chi/v5"
)

type TelegramSender interface {
	Send(ctx context.Context, text string) error
	OnAppEvent(ctx context.Context, ev notify.Event)
	Welcome(ctx context.Context) error
}

type Server struct {
	store *store.Store
	bot   TelegramSender
}

func New(s *store.Store, bot TelegramSender) http.Handler {
	api := &Server{store: s, bot: bot}
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/api/dashboard", api.dashboard)

	r.Get("/api/accounts", api.listAccounts)
	r.Post("/api/accounts", api.createAccount)
	r.Get("/api/accounts/{id}", api.getAccount)
	r.Put("/api/accounts/{id}", api.updateAccount)
	r.Delete("/api/accounts/{id}", api.deleteAccount)
	r.Post("/api/accounts/{id}/adjust", api.adjustAccount)

	r.Get("/api/projects", api.listProjects)
	r.Post("/api/projects", api.createProject)
	r.Get("/api/projects/{id}", api.getProject)
	r.Put("/api/projects/{id}", api.updateProject)
	r.Delete("/api/projects/{id}", api.deleteProject)
	r.Post("/api/projects/{id}/contribute", api.contribute)
	r.Post("/api/projects/{id}/withdraw", api.withdraw)
	r.Post("/api/projects/{id}/convert-asset", api.convertAsset)
	r.Post("/api/projects/{id}/items", api.createItem)
	r.Put("/api/projects/{id}/items/{itemId}", api.updateItem)
	r.Delete("/api/projects/{id}/items/{itemId}", api.deleteItem)
	r.Post("/api/projects/{id}/items/{itemId}/pay", api.payItem)

	r.Get("/api/assets", api.listAssets)
	r.Post("/api/assets", api.createAsset)
	r.Put("/api/assets/{id}", api.updateAsset)
	r.Delete("/api/assets/{id}", api.deleteAsset)

	r.Get("/api/transactions", api.listTransactions)
	r.Post("/api/transactions", api.createTransaction)
	r.Delete("/api/transactions/{id}", api.deleteTransaction)

	r.Get("/api/debts", api.listDebts)
	r.Post("/api/debts", api.createDebt)
	r.Get("/api/debts/{id}", api.getDebt)
	r.Put("/api/debts/{id}", api.updateDebt)
	r.Delete("/api/debts/{id}", api.deleteDebt)
	r.Post("/api/debts/{id}/pay", api.payDebt)
	r.Post("/api/debts/{id}/installments/{instId}/pay", api.payInstallment)

	r.Get("/api/telegram", api.telegramStatus)
	r.Post("/api/telegram/test", api.telegramTest)
	r.Get("/api/rates", api.rates)
	return r
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.Dashboard(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListAccounts(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetAccount(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var in models.AccountInput
	if !decodeBody(w, r, &in) {
		return
	}
	item, err := s.store.CreateAccount(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvAccountNew(item))
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request) {
	var in models.AccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.UpdateAccount(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvAccountUpd(item))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, _ := s.store.GetAccount(r.Context(), id)
	if err := s.store.DeleteAccount(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvAccountDel(item.Name))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) adjustAccount(w http.ResponseWriter, r *http.Request) {
	var in models.AdjustInput
	if !decodeBody(w, r, &in) {
		return
	}
	item, err := s.store.AdjustAccount(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvAdjust(item))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListProjects(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetProject(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var in models.ProjectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.CreateProject(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvProjectNew(item))
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	var in models.ProjectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.UpdateProject(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvProjectUpd(item))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, _ := s.store.GetProject(r.Context(), id)
	if err := s.store.DeleteProject(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvProjectDel(item.Name))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) contribute(w http.ResponseWriter, r *http.Request) {
	var in models.ContributeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.Contribute(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvProjectPay(item.Name, in.Amount))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) withdraw(w http.ResponseWriter, r *http.Request) {
	var in models.ContributeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.WithdrawProject(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvRefund(item.Name, in.Amount))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) convertAsset(w http.ResponseWriter, r *http.Request) {
	var in models.ConvertAssetInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.ConvertProjectToAsset(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvConvert(item.Name, item.Value))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createItem(w http.ResponseWriter, r *http.Request) {
	var in models.ItemInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.CreateItem(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.Event{Icon: "🧩", EN: "Sub-cost added\n" + item.Name, DE: "Teilkosten hinzugefügt\n" + item.Name})
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateItem(w http.ResponseWriter, r *http.Request) {
	var in models.ItemInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.UpdateItem(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "itemId"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.Event{Icon: "🧩", EN: "Sub-cost updated\n" + item.Name, DE: "Teilkosten aktualisiert\n" + item.Name})
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteItem(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "itemId")); err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.Event{Icon: "🗑", EN: "Sub-cost deleted", DE: "Teilkosten gelöscht"})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) payItem(w http.ResponseWriter, r *http.Request) {
	var in models.PayInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.PayItem(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "itemId"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvProjectPay(item.Name, in.Amount))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) listAssets(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListAssets(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createAsset(w http.ResponseWriter, r *http.Request) {
	var in models.AssetInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.CreateAsset(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvAssetNew(item))
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateAsset(w http.ResponseWriter, r *http.Request) {
	var in models.AssetInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.UpdateAsset(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvAssetUpd(item))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteAsset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, _ := s.store.GetAsset(r.Context(), id)
	if err := s.store.DeleteAsset(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvAssetDel(item.Name))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.store.ListTransactions(r.Context(), 200, q.Get("type"), q.Get("account_id"), q.Get("project_id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createTransaction(w http.ResponseWriter, r *http.Request) {
	var in models.TransactionInput
	if !decodeBody(w, r, &in) {
		return
	}
	item, err := s.store.CreateTransaction(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvTx(item))
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteTransaction(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvTxDeleted())
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listDebts(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListDebts(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getDebt(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetDebt(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createDebt(w http.ResponseWriter, r *http.Request) {
	var in models.DebtInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.CreateDebt(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvDebtNew(item))
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateDebt(w http.ResponseWriter, r *http.Request) {
	var in models.DebtInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.UpdateDebt(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvDebtUpd(item))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteDebt(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, _ := s.store.GetDebt(r.Context(), id)
	if err := s.store.DeleteDebt(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvDebtDel(item.Name))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) payDebt(w http.ResponseWriter, r *http.Request) {
	var in models.PayInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.PayDebt(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvDebtPay(item.Name, in.Amount, item.Remaining))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) payInstallment(w http.ResponseWriter, r *http.Request) {
	var in models.PayInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	item, err := s.store.PayInstallment(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "instId"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	s.emit(notify.EvDebtPay(item.Name, in.Amount, item.Remaining))
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) telegramStatus(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.TelegramSettings(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) telegramTest(w http.ResponseWriter, r *http.Request) {
	if s.bot == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "telegram disabled"})
		return
	}
	err := s.bot.Welcome(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "sent"})
}

func (s *Server) emit(ev notify.Event) {
	if s.bot == nil {
		return
	}
	go s.bot.OnAppEvent(context.Background(), ev)
}

func (s *Server) rates(w http.ResponseWriter, r *http.Request) {
	out, err := rates.Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "rates unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		msg := "invalid json"
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "int64") || strings.Contains(low, "overflow") {
			msg = "invalid: amount too large"
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, store.ErrInsufficient):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "insufficient"})
	case errors.Is(err, store.ErrInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}
