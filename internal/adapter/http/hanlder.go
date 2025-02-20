package http

//import (
//	"encoding/json"
//	"net/http"
//	"password-cracker/internal/core/domain"
//	"password-cracker/internal/ports"
//)
//
//type Handler struct {
//	crackerService ports.CrackerService
//}
//
//func NewHandler(service ports.CrackerService) *Handler {
//	return &Handler{
//		crackerService: service,
//	}
//}
//
//type StartCrackingRequest struct {
//	Hash     string                  `json:"hash"`
//	Settings domain.CrackingSettings `json:"settings"`
//}
//
//func (h *Handler) StartCracking(w http.ResponseWriter, r *http.Request) {
//	var req StartCrackingRequest
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		http.Error(w, "Invalid request body", http.StatusBadRequest)
//		return
//	}
//
//	job, err := h.crackerService.StartCracking(r.Context(), req.Hash, req.Settings)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//
//	json.NewEncoder(w).Encode(job)
//}
//
//func (h *Handler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
//	jobID := r.URL.Query().Get("id")
//	if jobID == "" {
//		http.Error(w, "Job ID is required", http.StatusBadRequest)
//		return
//	}
//
//	job, err := h.crackerService.GetJobStatus(r.Context(), jobID)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//
//	json.NewEncoder(w).Encode(job)
//}
//
//func (h *Handler) StopCracking(w http.ResponseWriter, r *http.Request) {
//	jobID := r.URL.Query().Get("id")
//	if jobID == "" {
//		http.Error(w, "Job ID is required", http.StatusBadRequest)
//		return
//	}
//
//	if err := h.crackerService.StopCracking(r.Context(), jobID); err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//
//	w.WriteHeader(http.StatusOK)
//}
