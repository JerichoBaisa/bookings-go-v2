package render

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

// RenderTemplate renders template using html/template
func RenderTemplateTEST1(w http.ResponseWriter, tmpl string) {
	parsetemplate, _ := template.ParseFiles("./templates/"+tmpl, "./templates/base.layout.html")
	err := parsetemplate.Execute(w, nil)
	if err != nil {
		fmt.Println("error parsing template", err)
		return
	}
}

// First approach
var tc1 = make(map[string]*template.Template)

func RenderTemplate1(w http.ResponseWriter, t string) {
	var tmpl *template.Template
	var err error

	_, inMap := tc1[t]
	if !inMap {
		// need t create the template
		log.Println("create template and adding to cache")
		err = createTemplateCache1(t)
		if err != nil {
			log.Println(err)
		}
	} else {
		// we have template in the cache
		log.Println("using cached template")
	}

	tmpl = tc1[t]
	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
	}

}

func createTemplateCache1(t string) error {
	templates := []string{
		fmt.Sprintf("./templates/%s", t),
		"./templates/base.layout.html",
	}

	// parse the template
	tmpl, err := template.ParseFiles(templates...)
	if err != nil {
		return err
	}

	// add template to cache
	tc1[t] = tmpl

	return nil

}
