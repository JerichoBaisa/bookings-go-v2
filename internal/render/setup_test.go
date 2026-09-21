package render

import (
	"encoding/gob"
	"net/http"
	"os"
	"testing"
	"time"

	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/config"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/models"
	"github.com/alexedwards/scs/v2"
)

var session *scs.SessionManager
var testApp config.AppConfig

func TestMain(m *testing.M) {

	gob.Register(models.Reservation{})

	// change this to true in production
	testApp.InProduction = false

	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = false

	testApp.Session = session

	app = &testApp

	os.Exit(m.Run())
}

// http.ResponseWriter is not exist in httpTest
// Create an object of interface to satisfy the condition where we can use the http.ResponseWriter
// no member, just a empty struct
// Then create the 3 methods where we can statisfy the http.ResponseWriter
// Header, WriteHeader and Write
type myHttpWriter struct{}

func (tw *myHttpWriter) Header() http.Header {
	var h http.Header
	return h
}

func (tw *myHttpWriter) Write(b []byte) (int, error) {
	length := len(b)
	return length, nil
}

func (tw *myHttpWriter) WriteHeader(statusCode int) {

}
