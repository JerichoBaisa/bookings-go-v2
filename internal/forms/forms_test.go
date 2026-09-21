package forms

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestForm_Valid name format is like tis because func valid is reference to a form
func TestForm_Valid(t *testing.T) {
	r := httptest.NewRequest("POST", "/whatever", nil)
	form := New(r.Form)

	isValid := form.Valid()

	if !isValid {
		t.Error("got invalid when should have been valid")
	}
}

func TestForm_Required(t *testing.T) {
	r := httptest.NewRequest("POST", "/whatever", nil)
	form := New(r.Form)

	form.Required("first_name", "last_name", "email", "phone")

	if form.Valid() {
		t.Error("form show valid when required fields missing")
	}

	postedData := url.Values{}
	postedData.Add("first_name", "Jericho")
	postedData.Add("last_name", "Baisa")
	postedData.Add("email", "jb@gmail.com")
	postedData.Add("phone", "555-555-5555")

	r, _ = http.NewRequest("POST", "/whatever", nil)

	r.PostForm = postedData

	form = New(r.PostForm)
	form.Required("first_name", "last_name", "email", "phone")

	if !form.Valid() {
		t.Error("show does not have required fields when it does")
	}

}

func TestForm_IsEmail(t *testing.T) {

	r, _ := http.NewRequest("POST", "/whatever", nil)

	form := New(r.PostForm)
	form.IsEmail("email")

	if form.Valid() {
		t.Error("show email valid when it does not")
	}

	postedData := url.Values{}
	postedData.Add("email", "adfasdfdasf@gmail.com")
	form = New(postedData)
	form.IsEmail("email")

	if !form.Valid() {
		t.Error("show email invalid when it does")
	}
}

func TestForm_MinLength(t *testing.T) {

	postedData := url.Values{}
	form := New(postedData)

	form.MinLength("xxx_last_name_xxx", 22)
	if form.Valid() {
		t.Error("shows min length for non-existing field")
	}

	isError := form.Errors.Get("xxx_last_name_xxx")
	if isError == "" {
		t.Error("should have and error, but none")
	}

	postedData = url.Values{}
	postedData.Add("last_name", "Baisa")
	form = New(postedData)

	ml := 10
	form.MinLength("last_name", ml)
	if form.Valid() {
		t.Error("shows min length 10 met where data is shorter")
	}

	postedData = url.Values{}
	postedData.Add("last_name", "Baisa")
	form = New(postedData)

	ml = 4
	form.MinLength("last_name", ml)
	if !form.Valid() {
		t.Error("shows min length 4 is not met whe it is")
	}

	isError = form.Errors.Get("last_name")
	if isError != "" {
		t.Error("should not have and error, but got one")
	}

}

func TestForm_Has(t *testing.T) {

	postedData := url.Values{}
	form := New(postedData)

	form.Has("phone")
	if form.Valid() {
		t.Error("form shows has field when it does not")
	}

	postedData = url.Values{}
	postedData.Add("phone", "555-555-5555")
	form = New(postedData)

	form.Has("phone")
	if !form.Valid() {
		t.Error("form shows does not have field when it should")
	}

}
