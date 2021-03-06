package handy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestInterceptorHandler struct {
	DefaultHandler
}

type DummyInterceptor struct{}

func (i *DummyInterceptor) Before(w http.ResponseWriter, r *http.Request) {
	for j := 0; j < 10000; j++ {
	}
}

func (i *DummyInterceptor) After(w http.ResponseWriter, r *http.Request) {
	for j := 0; j < 10000; j++ {
	}
}

func (t *TestInterceptorHandler) Interceptors() InterceptorChain {
	c := NewInterceptorChain()
	for i := 0; i < 20; i++ {
		c = c.Chain(new(DummyInterceptor))
	}

	return c
}

func BenchmarkInterceptorExecution(b *testing.B) {
	mux := NewHandy()
	mux.Handle("/foo", func() Handler {
		return new(TestInterceptorHandler)
	})

	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		b.Fatal(err)
	}

	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, req)
	}
}
