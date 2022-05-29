package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/airheartdev/replicache"
	"github.com/airheartdev/replicache/memory"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/r3labs/sse/v2"
)

type Todo struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
	Sort      int    `json:"sort"`
}

func (t Todo) String() string {
	b, _ := json.Marshal(t)
	return string(b)
}

type UpdateTodo struct {
	ID      string `json:"id"`
	Changes Todo   `json:"changes"`
}

type CompleteTodos struct {
	IDs       []string `json:"id"`
	Completed bool     `json:"completed"`
}

func main() {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(cors.Handler(cors.Options{
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", replicache.ReplicacheRequestIDHeader},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	events := sse.New()
	router.Get("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		events.ServeHTTP(w, r)
	})
	events.CreateStream("mutations")

	be := memory.New[Todo]()
	// be.PutEntry("3s3rnj", "todo/ticker", Todo{
	// 	ID:        "ticket",
	// 	Text:      "Ticker",
	// 	Completed: false,
	// 	Sort:      0,
	// }, 1)
	lock := &sync.Mutex{}

	rep := replicache.New[Todo](replicache.WithAuth(func(ctx context.Context, token string) bool {
		// log.Println("Auth", token)
		return true
	}))

	pullHandler := rep.HandlePull(func(pr *replicache.PullRequest, spaceID string) (replicache.PullResponse[Todo], error) {
		// Poor man's transaction
		lock.Lock()
		defer lock.Unlock()

		lastMutationID, ok := be.GetLastMutationID(pr.ClientID)
		if !ok {
			lastMutationID = 0
		}
		responseCookie, ok := be.GetCookie(spaceID)
		if !ok {
			responseCookie = 0
		}

		resp := replicache.PullResponse[Todo]{
			LastMutationID: lastMutationID,
			Cookie:         responseCookie,
			Patch:          []replicache.PatchOperation[Todo]{},
		}

		if pr.Cookie == 0 {
			resp.Patch = append(resp.Patch, replicache.PatchOperation[Todo]{
				Op: replicache.PatchClear,
			})
		}

		entries := be.GetChangedEntries(spaceID, pr.Cookie)

		for _, entry := range entries {
			key := todoKey(entry.Key)
			if entry.Deleted {
				resp.Patch = append(resp.Patch, replicache.PatchOperation[Todo]{
					Op:  replicache.PatchDel,
					Key: &key,
				})
			} else {
				resp.Patch = append(resp.Patch, replicache.PatchOperation[Todo]{
					Op:    replicache.PatchPut,
					Key:   &key,
					Value: &entry.Value,
				})
			}
		}

		return resp, nil
	})

	pushHandler := rep.HandlePush(func(pr *replicache.PushRequest, spaceID string) error {
		lock.Lock()
		defer lock.Unlock()

		prevVersion, ok := be.GetCookie(spaceID)
		if !ok {
			prevVersion = 0
		}

		nextVersion := prevVersion + 1
		lastMutationID, ok := be.GetLastMutationID(pr.ClientID)
		if !ok {
			lastMutationID = 0
		}

		// log.Printf("nextVersion: %d", nextVersion)
		// log.Printf("lastMutationID: %d", lastMutationID)

		tx := memory.ReplicacheTransaction[Todo](be, spaceID, pr.ClientID, nextVersion)

		for _, mut := range pr.Mutations {
			expectedMutationID := lastMutationID + 1
			if mut.ID < expectedMutationID {
				log.Printf("Mutation %d has already been processed - skipping ", mut.ID)
				continue
			}

			if mut.ID > expectedMutationID {
				log.Printf("Mutation %d is from the future - aborting", mut.ID)
				break
			}

			log.Printf("Processing mutation (%s): %s", mut.Name, string(mut.Args))

			switch mut.Name {
			case "putTodo":
				newTodo := new(Todo)
				err := json.Unmarshal(mut.Args, newTodo)
				if err != nil {
					log.Printf("Error unmarshalling putTodo(): %s", err)
					return err
				}

				tx.Put(todoKey(newTodo.ID), newTodo)

				// err = be.PutEntry(spaceID, todoKey(newTodo.ID), newTodo, nextVersion)
				// if err != nil {
				// 	log.Printf("Error putting entry: %s", err)
				// 	return err
				// }

			case "updateTodo":
				update := new(UpdateTodo)
				err := json.Unmarshal(mut.Args, update)
				if err != nil {
					log.Printf("Error unmarshalling updateTodo(): %s", err)
					return err
				}

				// log.Printf("Update todo: %s", update.ID)

				todo, err := tx.Get(update.ID)
				if err != nil {
					return err
				}

				// value, err := be.GetEntry(spaceID, todoKey(update.ID))
				// if err != nil {
				// 	return err
				// }

				if todo.Completed != update.Changes.Completed {
					todo.Completed = update.Changes.Completed
				}
				if todo.Sort != update.Changes.Sort {
					todo.Sort = update.Changes.Sort
				}
				if todo.Text != update.Changes.Text && update.Changes.Text != "" {
					todo.Text = update.Changes.Text
				}

				tx.Put(todoKey(todo.ID), todo)

				// be.PutEntry(spaceID, todoKey(update.ID), todo, nextVersion)

			case "deleteTodos":
				ids := []string{}
				err := json.Unmarshal(mut.Args, &ids)
				if err != nil {
					log.Printf("Error unmarshalling updateTodo(): %s - '%s'", err, string(mut.Args))
					return err
				}
				for _, id := range ids {
					tx.Del(todoKey(id))
				}

			case "completeTodos":
				change := new(CompleteTodos)
				err := json.Unmarshal(mut.Args, change)
				if err != nil {
					log.Printf("Error unmarshalling updateTodo(): %s", err)
					return err
				}

				for _, id := range change.IDs {
					todo, err := tx.Get(todoKey(id))
					if err != nil {
						return err
					}

					todo.Completed = change.Completed
					tx.Put(todoKey(id), todo)
				}
			}

			lastMutationID = expectedMutationID
		}

		be.SetLastMutationID(pr.ClientID, lastMutationID)
		be.SetCookie(spaceID, nextVersion)
		tx.Flush()

		events.Publish("mutations", &sse.Event{
			Data: []byte("ping"),
		})

		return nil
	})

	router.Post(replicache.DefaultPullEndpoint, pullHandler)
	router.Post(replicache.DefaultPushEndpoint, pushHandler)

	log.Println("Listening on http://localhost:1234")
	log.Fatal(http.ListenAndServe("127.0.0.1:1234", router))
}

func todoKey(id string) string {
	return fmt.Sprintf("todo/%s", id)
}

func idFromKey(key string) string {
	return key[5:]
}
