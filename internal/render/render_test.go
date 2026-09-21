package render

import (
	"net/http"
	"testing"

	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/models"
)



func TestAddDefaultData(t *testing.T) {
	var td models.TemplateData

	r, err := getSession()
	if err != nil {
		t.Error(err)
	}

	// Need to test ex: flash
	session.Put(r.Context(), "flash", "123")

	result := AddDefaultData(&td, r)
	if result.Flash != "123" {
		t.Error("flash value of 123 not found in the session")
	}

}

func TestRenderTemplate(t *testing.T) {
	pathToTemplates = "./../../templates"
	tc, err := CreateTemplateCache()
	if err != nil {
		t.Error(err)
	}

	app.TemplateCache = tc
	// app.UseCache = true //
	r, err := getSession()
	if err != nil {
		t.Error(err)
	}

	var mhw myHttpWriter

	err = Template(&mhw, r, "home.page.html", &models.TemplateData{})
	if err != nil {
		t.Error("error writing template to browser from test render templates")
	}

	err = Template(&mhw, r, "non-existent.page.html", &models.TemplateData{})
	if err == nil {
		t.Error("rendered template the does not exist from test render templates")
	}

}

func getSession() (*http.Request, error) {
	r, err := http.NewRequest("GET", "/some-url", nil)
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	ctx, _ = session.Load(ctx, r.Header.Get("X-Session"))
	r = r.WithContext(ctx)

	return r, nil

}

func TestNewTemplates(t *testing.T) {
	NewRenderer(app)
}

func TestCreateTemplateCache(t *testing.T) {
	pathToTemplates = "./../../templates"
	_, err := CreateTemplateCache()
	if err != nil {
		t.Error(err)
	}
}
