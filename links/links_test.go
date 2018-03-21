// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package links

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"firebase.google.com/go/internal"
	"google.golang.org/api/option"
)

var client *Client
var testLinkStatsResponse []byte

func TestMain(m *testing.M) {
	defaultTestOpts := []option.ClientOption{
		option.WithTokenSource(&internal.MockTokenSource{
			AccessToken: "test-token",
		})}

	var err error
	client, err = NewClient(context.Background(), defaultTestOpts...)
	if err != nil {
		log.Fatalln(err)
	}

	testLinkStatsResponse, err = ioutil.ReadFile("../testdata/get_link_stats.json")
	if err != nil {
		log.Fatalln(err)
	}
	os.Exit(m.Run())
}

func TestCreateEventStatsMarshal(t *testing.T) {
	es := &EventStats{Platform: Desktop, ET: AppFirstOpen, Count: 4}
	m, err := json.Marshal(es)
	if err != nil {
		t.Error(err)
	}
	if string(m) != `{"platform":"DESKTOP","event":"APP_FIRST_OPEN","count":"4"}` {
		t.Errorf(`Marshal(%v) = %v, expecting: {"platform":"DESKTOP","event":"APP_FIRST_OPEN","count":4}`,
			es, string(m))
	}
}

func TestCreateEventStatsString(t *testing.T) {
	es := EventStats{Platform: IOS, ET: AppReOpen, Count: 4}

	want := "{IOS APP_RE_OPEN 4}"
	if str := fmt.Sprintf("%v", es); str != want {
		t.Errorf("String representation of EventStats, got: %q, want: %q", str, want)
	}
}

func TestReadJSON(t *testing.T) {
	var ls LinkStats
	err := json.Unmarshal(testLinkStatsResponse, &ls)
	if err != nil {
		log.Fatalln(err)
	}
	want := getWantedStatResult()
	if !reflect.DeepEqual(ls, want) {
		t.Errorf("read json file, got %#v, want: %#v", ls, want)
	}
}

func TestGetLinksRequest(t *testing.T) {
	var tr *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr = r
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testLinkStatsResponse))
	}))
	defer ts.Close()

	client.linksEndpoint = ts.URL

	_, err := client.LinkStats(context.Background(), "https://mock", StatOptions{DurationDays: 7})
	if err != nil {
		t.Fatal(err)
	}

	wantRequestURI := "/https%3A%2F%2Fmock/linkStats?durationDays=7"
	if tr.RequestURI != wantRequestURI {
		t.Errorf("expecting RequestURI: %q, got %q", tr.RequestURI, wantRequestURI)
	}
}
func TestGetLinksStats(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(testLinkStatsResponse))
	}))
	defer ts.Close()

	client.linksEndpoint = ts.URL

	ls, err := client.LinkStats(context.Background(), "https://mock", StatOptions{DurationDays: 7})
	if err != nil {
		t.Fatal(err)
	}
	wantLinkStats := getWantedStatResult()

	if !reflect.DeepEqual(*ls, wantLinkStats) {
		t.Errorf("read json file, got %#v, want: %#v", *ls, wantLinkStats)
	}
}

func TestGetLinksStatsServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte("intentional error"))
	}))
	defer ts.Close()

	client.linksEndpoint = ts.URL

	_, err := client.LinkStats(context.Background(), "https://mock", StatOptions{DurationDays: 7})
	we := "http error status: 500; reason: intentional error"
	if err == nil || err.Error() != we {
		t.Fatalf("got error: %q, want: %q", err, we)
	}
}
func TestInvalidShortLink(t *testing.T) {
	_, err := client.LinkStats(context.Background(), "asdf", StatOptions{DurationDays: 2})
	we := "short link must start with `https://`"
	if err == nil || err.Error() != we {
		t.Errorf("LinkStats(<invalid short link>) err = %q, wanted err = %q", err, we)
	}
}

func TestInvalidDurationDays(t *testing.T) {
	_, err := client.LinkStats(context.Background(), "https://mock", StatOptions{DurationDays: -1})
	we := "durationDays must be > 0"
	if err == nil || err.Error() != we {
		t.Errorf("LinkStats(<invalid durationDays) err = %q, wanted err = %q", err, we)
	}
}

func TestPlatformUnmarshalError(t *testing.T) {
	var p Platform
	if err := p.UnmarshalJSON([]byte("")); err == nil {
		t.Errorf("expecting Unmarshall failure ")
	}
	we := `unknown platform "bla"`
	if err := p.UnmarshalJSON([]byte(`"bla"`)); err == nil || err.Error() != we {
		t.Errorf("Unmarshall(bla):%q; want:%q", err, we)
	}

}
func TestEventUnmarshalError(t *testing.T) {
	var e EventType
	if err := e.UnmarshalJSON([]byte("")); err == nil {
		t.Errorf("expecting Unmarshall failure ")
	}
	we := `unknown event type "bla"`
	if err := e.UnmarshalJSON([]byte(`"bla"`)); err == nil || err.Error() != we {
		t.Errorf("Unmarshall(bla):%q; want:%q", err, we)
	}
}
func getWantedStatResult() LinkStats {
	return LinkStats{
		EventStats: []EventStats{
			{
				Platform: Android,
				Count:    123,
				ET:       Click,
			},
			{
				Platform: IOS,
				Count:    123,
				ET:       Click,
			},
			{
				Platform: Desktop,
				Count:    456,
				ET:       Click,
			},
			{
				Platform: Android,
				Count:    99,
				ET:       AppInstall,
			},
			{
				Platform: Android,
				Count:    42,
				ET:       AppFirstOpen,
			},
			{
				Platform: Android,
				Count:    142,
				ET:       AppReOpen,
			},
			{
				Platform: IOS,
				Count:    124,
				ET:       Redirect,
			},
		},
	}
}