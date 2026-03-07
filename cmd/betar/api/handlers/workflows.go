package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/asabya/betar/internal/workflow"
	"github.com/asabya/betar/pkg/types"
	"github.com/gorilla/mux"
)

// RegisterWorkflowHandlers registers the /workflows HTTP endpoints on the
// given router.  The orchestrator must not be nil.
func RegisterWorkflowHandlers(r *mux.Router, orch *workflow.Orchestrator) {
	h := &workflowHandler{orch: orch}
	r.HandleFunc("/workflows", h.createWorkflow).Methods("POST")
	r.HandleFunc("/workflows", h.listWorkflows).Methods("GET")
	r.HandleFunc("/workflows/{id}", h.getWorkflow).Methods("GET")
	r.HandleFunc("/workflows/{id}", h.cancelWorkflow).Methods("DELETE")
}

type workflowHandler struct {
	orch *workflow.Orchestrator
}

func (h *workflowHandler) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var def types.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	wf, err := h.orch.CreateWorkflow(r.Context(), def)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Run the workflow asynchronously so it survives HTTP timeouts.
	// Use a detached context — the workflow must not be canceled when
	// the HTTP request completes.
	if err := h.orch.RunWorkflowAsync(context.Background(), wf.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (h *workflowHandler) listWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows, err := h.orch.ListWorkflows(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

func (h *workflowHandler) getWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	wf, err := h.orch.GetWorkflow(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (h *workflowHandler) cancelWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.orch.CancelWorkflow(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	wf, err := h.orch.GetWorkflow(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}
