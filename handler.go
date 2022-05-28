package replicache

import (
	"encoding/json"
	"log"
	"net/http"
)

const DefaultPushEndpoint = "/replicache-push"
const DefaultPullEndpoint = "/replicache-pull"
const applicationJSON = "application/json"
const ReplicacheRequestIDHeader = "X-Replicache-RequestID"
const authorizationHeader = "Authorization"

func (r *Replicache[T]) HandlePush(fn func(pr *PushRequest, spaceID string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if !validateRequest(w, req, r.options.authFn) {
			return
		}

		push := new(PushRequest)
		err := json.NewDecoder(req.Body).Decode(push)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		spaceID := req.URL.Query().Get("spaceID")
		err = fn(push, spaceID)
		if err != nil {
			log.Printf("Push Error: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (r *Replicache[T]) HandlePull(fn func(pr *PullRequest, spaceID string) (PullResponse[T], error)) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if !validateRequest(w, req, r.options.authFn) {
			return
		}

		pull := new(PullRequest)
		err := json.NewDecoder(req.Body).Decode(pull)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		spaceID := req.URL.Query().Get("spaceID")

		resp, err := fn(pull, spaceID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", applicationJSON)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func validateRequest(w http.ResponseWriter, r *http.Request, authFn AuthFn) bool {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}

	if r.Header.Get("Content-Type") != applicationJSON {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	if requestID := r.Header.Get(ReplicacheRequestIDHeader); requestID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	if authFn != nil {
		auth := r.Header.Get(authorizationHeader)
		if !authFn(r.Context(), auth) {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}
	}

	return true
}
