package main

import (
	"html/template"
)

/* Templates */
var registerTemplate = template.Must(template.ParseFiles("./html/register.html"))
var loginTemplate = template.Must(template.ParseFiles("./html/login.html"))
var flashTemplate = template.Must(template.ParseFiles("./html/index.html"))

var TemplateSet = template.Must(template.ParseFiles("./templates/templates.templ"))

type TemplateInput struct {
	Usr  User
	Card UserCard
}

type PrefInput struct {
	Username    string
	Preferences []Preference
}

type UserInput struct {
	Username string
}

func ReloadTemplates() {
	TemplateSet = template.Must(template.ParseFiles("./templates/templates.templ"))
}
