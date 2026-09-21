package models

import "bitbucket.org/JerichoBaisa/bookings-go-v2/internal/forms"

// TemplateData holds data sent from handlers to templates
type TemplateData struct {
	StringMap       map[string]string
	IntMap          map[string]int
	FloatMap        map[string]float32
	Data            map[string]interface{}
	CSRFToken       string
	Flash           string //flash message
	Warning         string
	Error           string
	Form            *forms.Form
	IsAuthenticated int
}
