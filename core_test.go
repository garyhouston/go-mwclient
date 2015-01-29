package mwclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"cgt.name/pkg/go-mwclient/params"
)

func setup(handler func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(handler))
	client, err := New(server.URL, "go-mwclient test")
	if err != nil {
		panic(err)
	}

	return server, client
}

func TestLogin(t *testing.T) {
	loginHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			t.Fatal("Bad test parameters")
		}

		// This is really difficult to read. Sorry...
		// Possible errors: NoName, NeedToken, WrongToken, EmptyPass, WrongPass, NotExists
		if r.PostForm.Get("lgname") == "" {
			fmt.Fprint(w, `{"login":{"result":"NoName"}}`)
		} else if r.PostForm.Get("lgtoken") == "" {
			fmt.Fprint(w, `{"login":{"result":"NeedToken","token":"7aaaf636d99d46cf2656561c5d099ad7","cookieprefix":"dawiki","sessionid":"e9bffbfa38636ac9f550b5d37fb25d80"}}`)
		} else {
			if r.PostForm.Get("lgtoken") != "7aaaf636d99d46cf2656561c5d099ad7" {
				fmt.Fprint(w, `{"login":{"result":"WrongToken"}}`)
			} else {
				if r.PostForm.Get("lgpassword") == "" {
					fmt.Fprint(w, `{"login":{"result":"EmptyPass"}}`)
				} else {
					if r.PostForm.Get("lgname") == "username" {
						if r.PostForm.Get("lgpassword") == "password" {
							// success
							fmt.Fprint(w, `{"login":{"result":"Success","lguserid":1,"lgusername":"username","lgtoken":"7aaaf636d99d46cf2656561c5d099ad7","cookieprefix":"dawiki","sessionid":"e9bffbfa38636ac9f550b5d37fb25d80"}}`)
						} else {
							fmt.Fprint(w, `{"login":{"result":"WrongPass"}}`)
						}
					} else {
						fmt.Fprint(w, `{"login":{"result":"NotExists"}}`)
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	server, client := setup(loginHandler)
	defer server.Close()

	// good
	if err := client.Login("username", "password"); err != nil {
		t.Error("passed good login. expected no error, but received error:", err)
	}

	// bad
	if err := client.Login("", ""); err == nil {
		t.Error("passed empty user. expected NoName error, but no error received")
	} else {
		if apiErr, ok := err.(APIError); ok {
			if apiErr.Code != "NoName" {
				t.Errorf("expected NoName error, received: %v", apiErr.Code)
			} else {
				t.Log(apiErr.Code)
			}
		}
	}

	if err := client.Login("username", ""); err == nil {
		t.Error("passed empty password. expected EmptyPass error, but no error received")
	} else {
		if apiErr, ok := err.(APIError); ok {
			if apiErr.Code != "EmptyPass" {
				t.Errorf("expected EmptyPass error, received: %v", apiErr.Code)
			} else {
				t.Log(apiErr.Code)
			}
		}
	}

	if err := client.Login("badusername", "password"); err == nil {
		t.Error("passed bad user. expected NotExists error, but no error received")
	} else {
		if apiErr, ok := err.(APIError); ok {
			if apiErr.Code != "NotExists" {
				t.Errorf("expected NotExists error, received: %v", apiErr.Code)
			} else {
				t.Log(apiErr.Code)
			}
		}
	}

	if err := client.Login("username", "badpassword"); err == nil {
		t.Error("passed bad password. expected WrongPass error, but no error received")
	} else {
		if apiErr, ok := err.(APIError); ok {
			if apiErr.Code != "WrongPass" {
				t.Errorf("expected WrongPass error, received: %v", apiErr.Code)
			} else {
				t.Log(apiErr.Code)
			}
		}
	}
}

func TestLoginToken(t *testing.T) {
	loginHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			t.Fatal("Bad test parameters")
		}

		lgtoken := "7aaaf636d99d46cf2656561c5d099ad7"

		if r.PostForm.Get("lgtoken") == "" {
			fmt.Fprintf(w,
				`{"login":{"result":"NeedToken","token":"%s","cookieprefix":"dawiki","sessionid":"e9bffbfa38636ac9f550b5d37fb25d80"}}`,
				lgtoken)
		} else {
			if got := r.PostForm.Get("lgtoken"); got == lgtoken {
				fmt.Fprintf(w,
					`{"login":{"result":"Success","lguserid":1,"lgusername":"username","lgtoken":"%s","cookieprefix":"dawiki","sessionid":"e9bffbfa38636ac9f550b5d37fb25d80"}}`,
					lgtoken)
			} else {
				t.Fatalf("sent lgtoken '%s', got '%s'", lgtoken, got)
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	server, client := setup(loginHandler)
	defer server.Close()

	if err := client.Login("username", "password"); err != nil {
		t.Errorf("Login() returned err: %v", err)
	}
}

func TestMaxlagOn(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			t.Fatal("Bad test parameters")
		}

		if r.Form.Get("maxlag") == "" {
			t.Fatalf("maxlag param not set. Params: %s", r.Form.Encode())
		}
	}

	server, client := setup(httpHandler)
	defer server.Close()

	p := params.Values{}
	client.Maxlag.On = true
	client.call(p, false)
}

func TestMaxlagOff(t *testing.T) {
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			t.Fatal("Bad test parameters")
		}

		if r.Form.Get("maxlag") != "" {
			t.Fatalf("maxlag param set. Params: %s", r.Form.Encode())
		}
	}

	server, client := setup(httpHandler)
	defer server.Close()

	p := params.Values{}
	// Maxlag is off by default
	client.call(p, false)
}

func TestMaxlagRetryFail(t *testing.T) {
	// There's sleep calls in this test
	if testing.Short() {
		t.SkipNow()
	}

	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			t.Fatal("Bad test parameters")
		}
		if r.Form.Get("maxlag") == "" {
			t.Fatalf("maxlag param not set. Params: %s", r.Form.Encode())
		}

		header := w.Header()
		header.Set("X-Database-Lag", "10") // Value does not matter
		header.Set("Retry-After", "1")     // Value *does* matter
	}

	server, client := setup(httpHandler)
	defer server.Close()

	p := params.Values{}
	client.Maxlag.On = true
	_, err := client.call(p, false)
	if err != ErrAPIBusy {
		t.Fatalf("Expected ErrAPIBusy error from call(), got: %v", err)
	}
}
