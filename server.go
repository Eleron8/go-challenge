package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
)

// App represents the server's internal state.
// It holds configuration about providers and content
type App struct {
	ContentClients map[Provider]Client
	Config         ContentMix
}

// func (a App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")
// 	if req.Method != http.MethodGet {
// 		http.Error(w, "method not allowed. Use GET method", http.StatusMethodNotAllowed)
// 		return
// 	}
// 	log.Printf("%s %s", req.Method, req.URL.String())
// 	countStr := req.URL.Query().Get("count")

// 	offsetStr := req.URL.Query().Get("offset")

// 	count, err := strconv.Atoi(countStr)
// 	if err != nil {
// 		http.Error(w, "Bad request", http.StatusBadRequest)
// 		return
// 	}
// 	offset, err := strconv.Atoi(offsetStr)
// 	if err != nil {
// 		http.Error(w, "Bad request", http.StatusBadRequest)
// 		return
// 	}
// 	if offset > len(a.Config) {
// 		offset = offset % len(a.Config)
// 	}
// 	var res []*ContentItem

// 	for count > 0 {
// 		for _, v := range a.Config[offset:] {
// 			if count == 0 {
// 				break
// 			}
// 			var resp []*ContentItem
// 			resp, err = a.ContentClients[v.Type].GetContent(req.RemoteAddr, 1)
// 			if err != nil {
// 				resp, err = a.ContentClients[*v.Fallback].GetContent(req.RemoteAddr, 1)
// 				if err != nil {
// 					count = 0
// 					break
// 				}
// 			}
// 			res = append(res, resp...)
// 			count--

// 		}

// 		offset = 0
// 	}
// 	// resp, err := a.ContentClients[a.Config[0].Type].GetContent(req.RemoteAddr, count)
// 	// if err != nil {
// 	// 	http.Error(w, "Internal error", http.StatusInternalServerError)
// 	// 	return
// 	// }
// 	json.NewEncoder(w).Encode(res)

// }

type ItemCounter struct {
	contentItem *ContentItem
	counter     int
}

func (a App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed. Use GET method", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("%s %s", req.Method, req.URL.String())
	countStr := req.URL.Query().Get("count")

	offsetStr := req.URL.Query().Get("offset")

	count, err := strconv.Atoi(countStr)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if offset > len(a.Config) {
		offset = offset % len(a.Config)
	}
	res := make([]*ContentItem, count)
	var wg sync.WaitGroup
	wg.Add(count)
	primaryCount := count
	contentCh := make(chan ItemCounter, count)
	markerCh := make(chan int)
	go func() {
		wg.Wait()
		close(contentCh)
		close(markerCh)
	}()
	for count > 0 {
		for _, v := range a.Config[offset:] {
			if count == 0 {
				break
			}
			go func() {
				defer wg.Done()
				var resp *ContentItem
				c := <-markerCh
				resp, err = a.ContentClients[v.Type].GetContent(req.RemoteAddr, 1)
				if err != nil {
					if v.Fallback == nil {
						contentCh <- ItemCounter{
							contentItem: nil,
							counter:     primaryCount - c,
						}
						return
					}
					resp, err = a.ContentClients[*v.Fallback].GetContent(req.RemoteAddr, count)
					if err != nil {
						contentCh <- ItemCounter{
							contentItem: nil,
							counter:     primaryCount - c,
						}
					} else {
						contentCh <- ItemCounter{
							contentItem: resp,
							counter:     primaryCount - c,
						}
					}
				} else {
					contentCh <- ItemCounter{
						contentItem: resp,
						counter:     primaryCount - c,
					}
				}

			}()
			markerCh <- count
			count--
		}
		offset = 0
	}
	wg.Wait()

	for cont := range contentCh {
		res[cont.counter] = cont.contentItem

	}

	for i, v := range res {
		if v == nil {
			res = res[:i]
			break
		}
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, "Server error", http.StatusBadGateway)
		return
	}
}
