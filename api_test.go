package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	SimpleContentRequest = httptest.NewRequest("GET", "/?offset=0&count=5", nil)
	OffsetContentRequest = httptest.NewRequest("GET", "/?offset=5&count=5", nil)
)

func runRequest(t *testing.T, srv http.Handler, r *http.Request) (content []*ContentItem) {
	response := httptest.NewRecorder()
	srv.ServeHTTP(response, r)

	if response.Code != 200 {
		t.Fatalf("Response code is %d, want 200", response.Code)
		return
	}

	decoder := json.NewDecoder(response.Body)
	err := decoder.Decode(&content)
	if err != nil {
		t.Fatalf("couldn't decode Response json: %v", err)
	}

	return content
}

func TestResponseCount(t *testing.T) {
	content := runRequest(t, app, SimpleContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}
}

func TestResponseOrder(t *testing.T) {
	content := runRequest(t, app, SimpleContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

	for i, item := range content {
		if Provider(item.Source) != DefaultConfig[i%len(DefaultConfig)].Type {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, DefaultConfig[i].Type,
			)
		}
	}
}

func TestOffsetResponseOrder(t *testing.T) {
	content := runRequest(t, app, OffsetContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

	for j, item := range content {
		i := j + 5
		if Provider(item.Source) != DefaultConfig[i%len(DefaultConfig)].Type {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, DefaultConfig[i].Type,
			)
		}
	}
}

func TestFallback(t *testing.T) {
	type testConfig struct {
		configs        []ContentConfig
		request        *http.Request
		expectedLength int
		expectedOrder  []string
	}

	testcases := []testConfig{
		{
			configs: []ContentConfig{
				{
					Type:     Provider1,
					Fallback: &Provider2,
				},
				{
					Type:     Provider2,
					Fallback: nil,
				},
				{
					Type:     Provider4,
					Fallback: &Provider3,
				},
			},
			request:        SimpleContentRequest,
			expectedLength: 5,
			expectedOrder:  []string{"1", "2", "3", "1", "2"},
		},
		{
			configs: []ContentConfig{
				{
					Type:     Provider1,
					Fallback: &Provider2,
				},
				{
					Type:     Provider4,
					Fallback: nil,
				},
				{
					Type:     Provider4,
					Fallback: &Provider3,
				},
			},
			request:        OffsetContentRequest,
			expectedLength: 2,
			expectedOrder:  []string{"3", "1"},
		},
		{
			configs: []ContentConfig{
				{
					Type:     Provider1,
					Fallback: &Provider2,
				},
				{
					Type:     Provider2,
					Fallback: nil,
				},
				{
					Type:     Provider4,
					Fallback: &Provider3,
				},
				{
					Type:     Provider4,
					Fallback: &Provider4,
				},
			},
			request:        httptest.NewRequest("GET", "/?offset=7&count=5", nil),
			expectedLength: 0,
			expectedOrder:  []string{},
		},
	}

	for _, tt := range testcases {
		testApp := App{
			ContentClients: map[Provider]Client{
				Provider1: SampleContentProvider{Source: Provider1},
				Provider2: SampleContentProvider{Source: Provider2},
				Provider3: SampleContentProvider{Source: Provider3},
				Provider4: SampleContentProvider{Source: Provider4},
			},
			Config: tt.configs,
		}

		content := runRequest(t, testApp, tt.request)

		if len(content) != tt.expectedLength {
			t.Fatalf("Got %d items back, want %d", len(content), tt.expectedLength)
		}

		for i, v := range content {
			if v.Source != tt.expectedOrder[i] {
				t.Fatalf("Got items back from provider %s, want from provider %s", v.Source, tt.expectedOrder[i])
			}
		}
	}
}
