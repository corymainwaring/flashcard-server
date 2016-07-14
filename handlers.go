package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

func study(u *User, rw http.ResponseWriter, req *http.Request) {
	var templateName string
	if u.Preferences["written"].GetVal() == "true" {
		templateName = "written"
		// Feedback Mechanism
		switch req.FormValue("correct") {
		case "true":
			u.Cards[u.Cur].Correct++
			u.Cards[u.Cur].LastSeen = time.Now()
			u.Cur++
			break
		case "false":
			u.Cards[u.Cur].Wrong++
			u.Cards[u.Cur].LastSeen = time.Now()
			u.Cur++
			break
		}
	} else {
		templateName = "flashcard"
		switch req.FormValue("submit") {
		case "Correct":
			u.Cards[u.Cur].Correct++
			u.Cards[u.Cur].LastSeen = time.Now()
			u.Cur++
			break
		case "Incorrect":
			u.Cards[u.Cur].Wrong++
			u.Cards[u.Cur].LastSeen = time.Now()
			u.Cur++
			break
		}
	}

	// Define MIME type
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Prepare Flashcard
	if u.Cur > 19 {
		u.Cur = 0
		u.SortAndAddCards(mCards)
		if err := u.Save(); err != nil {
			log.Println(err)
		}
	}
	// Parse file through templater
	err := TemplateSet.ExecuteTemplate(rw, templateName, TemplateInput{*u, u.Cards[u.Cur]})
	if err != nil {
		log.Println("flashcardHandler:", err)
	}
}

func quitHandler(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(rw, "Quitting!")
	quit <- 1
	close(quit)
}

func seen(u *User, rw http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(rw, len(u.Cards))
}
func login(u *User, rw http.ResponseWriter, req *http.Request) {
	if req.FormValue("submit") == "Login" {
		// Logout running sessions
		for _, v := range UserList {
			if v != nil && v.Passphrase == strings.ToLower(req.FormValue("passphrase")) {
				logoutUser(v)
				u = v
				break
			}
		}
		var err error
		if u == nil {
			u, err = LoadUser(strings.ToLower(req.FormValue("passphrase")))
			if err != nil {
				logError(err, req)
				loginTemplate.Execute(rw, "That is not a valid name.")
				return
			}
		}
		/* USER LOGIN */
		id := genSessionId()
		u.Session = id
		UserList[id] = u
		rw.Header().Add("Set-Cookie", "session="+id)
		/* USER LOGIN */
		http.Redirect(rw, req, "."+req.RequestURI, 303)
	} else {
		loginTemplate.Execute(rw, nil)
	}
}

//TODO
func register(u *User, rw http.ResponseWriter, req *http.Request) {
	if req.FormValue("submit") == "Register" {
		if _, err := LoadUser(strings.ToLower(req.FormValue("passphrase"))); err == nil {
			registerTemplate.Execute(rw, "That name is already taken")
			return
		}
		uCard := make([]UserCard, 0, 20)
		for i, v := range mCards {
			uCard = append(uCard, UserCard{Master: v})
			if i == 19 {
				break
			}
		}
		id := genSessionId()
		u := CreateUser(strings.ToLower(req.FormValue("passphrase")), uCard, id)
		if err := u.Save(); err != nil {
			log.Println("registerHandler:", err)
		}
		/* USER LOGIN */
		UserList[id] = u
		rw.Header().Add("Set-Cookie", "session="+id)
		/* USER LOGIN */
		http.Redirect(rw, req, "/study", 303)
	} else {
		registerTemplate.Execute(rw, nil)
	}
}

func homeHandler(rw http.ResponseWriter, req *http.Request) {
	u, err := getUser(req)
	if err != nil {
		switch req.RequestURI {
		case "/register":
			register(u, rw, req)
		case "/login":
			login(u, rw, req)
		case "/study":
			login(u, rw, req)
		default:
			http.Redirect(rw, req, "/login", 303)
		}
		return
	}
	switch req.RequestURI {
	case "/study":
		study(u, rw, req)
	case "/seen":
		seen(u, rw, req)
	case "/cards":
		listFlashcards(u, rw, req)
	case "/pref":
		pref(u, rw, req)
	case "/logout":
		logout(u, rw, req)
	case "/login":
		http.Redirect(rw, req, "/study", 303)
	case "/add":
		personalAdd(u, rw, req)
	default:
		fourzerofour(u, rw, req)
	}
}

func logout(u *User, rw http.ResponseWriter, req *http.Request) {
	logoutUser(u)
	http.Redirect(rw, req, "/login", 303)
}

func reloadHandler(rw http.ResponseWriter, req *http.Request) {
	ReloadTemplates()
	http.Redirect(rw, req, "/study", 303)
}

func pref(u *User, rw http.ResponseWriter, req *http.Request) {
	prefs := DefaultPrefMap()
	if req.FormValue("submit") != "" {
		for _, v := range prefs {
			log.Println(req.FormValue(v.Name), v.Name)
			if req.FormValue(v.Name) != "" {
				v.SetVal(req.FormValue(v.Name))
				log.Println(v.GetVal())
			}
		}
		u.Preferences = prefs
		u.Save()
		//log.Println(u.Preferences["written"], "\n", prefs["written"])
	}

	/* Set Preference Template Inputs */
	inputPrefs := make([]Preference, 0, len(prefs))
	for _, v := range prefs {
		if u.Preferences[v.Name] != nil {
			v.SetVal(u.Preferences[v.Name].GetVal())
		}
		inputPrefs = append(inputPrefs, *v)
	}

	TemplateSet.ExecuteTemplate(rw, "preferences", PrefInput{Username: u.Passphrase, Preferences: inputPrefs})
}

func personalAdd(u *User, rw http.ResponseWriter, req *http.Request) {
	u, err := getUser(req)
	if err != nil {
		log.Println(time.Now(), " ", req.RemoteAddr, ": ", err)
		http.Redirect(rw, req, "/login", 303)
		return
	}
	if req.FormValue("submit") == "Add Flashcards" {
		inputArr := strings.Split(req.FormValue("input"), "\n")
		order := make([]int, 4)
		orderArr := strings.Split(inputArr[0], ";")
		for i := range orderArr {
			if orderArr[i] == "english" {
				order[0] = i
			} else if orderArr[i] == "simplified" {
				order[1] = i
			} else if orderArr[i] == "traditional" {
				order[2] = i
			} else if orderArr[i] == "pinyin" {
				order[3] = i
			}
		}
		if len(inputArr) > 1 {
			for _, v := range inputArr[1:] {
				rec := strings.Split(v, ";")
				// Check Existing
				existing := false
				for i := range mCards {
					if mCards[i].Simplified == rec[order[1]] {
						// Insert English
						mCards[i].English = append(mCards[i].English, strings.Split(rec[order[0]], "/")...)
						// Dedup english
						sort.Sort(sort.StringSlice(mCards[i].English))
						for k := len(mCards[i].English) - 1; k > 1; k-- {
							if mCards[i].English[k] == mCards[i].English[k-1] {
								if k == len(mCards[i].English)-1 {
									mCards[i].English = mCards[i].English[:k]
								} else {
									mCards[i].English = append(mCards[i].English[:k], mCards[i].English[k+1:]...)
								}
							}
						}
						// Check card in user's cards
						u.MaybeAddCard(mCards[i])
					}
				}
				// Not Existing
				if !existing {
					// Add to mCards
					mCards = append(mCards, MasterCard{
						Id:          len(mCards),
						Simplified:  rec[order[1]],
						Traditional: rec[order[2]],
						Pinyin:      rec[order[3]],
						English:     strings.Split(rec[order[0]], "/"),
					})
					// Add to user's cards
					u.AddCard(mCards[len(mCards)-1])
				}
			}
		}
	}
	_ = u

	TemplateSet.ExecuteTemplate(rw, "add_personal", UserInput{Username: u.Passphrase})
}

func listFlashcards(u *User, rw http.ResponseWriter, req *http.Request) {
	for _, v := range u.Cards {
		fmt.Fprintln(rw, v)
		fmt.Fprintln(rw, v.Points())
	}
}

// TODO: Make a 404 page
func fourzerofour(u *User, rw http.ResponseWriter, req *http.Request) {
	fmt.Fprint(rw, req.RequestURI+" not found.")
}

/*func testAllTheCards(rw http.ResponseWriter, req *http.Request) {
	u, err := getUser(req)
	if err != nil {
		log.Println(time.Now(), " ", req.RemoteAddr, ": ", err)
		http.Redirect(rw, req, "/login", 303)
		return
	}
	k := 0
	for len(u.Cards) < len(mCards) {
		k++
		for i := range u.Cards {
			u.Cards[i].Correct = 1000
			u.Cards[i].LastSeen = time.Now()
			u.Cards[i].Wrong = 1
		}
		u.SortAndAddCards(mCards)
	}
	log.Println("Cards in Userfile: ", len(u.Cards))
	log.Println("Iterations: ", k)
}*/
