package main

import (
	"encoding/gob"
	"log"
	"os"
	"sort"
)

const (
	UserSavePath     = "./users/"
	PrefTypeSingle   = "radio"
	PrefTypeMultiple = "checkbox"
)

type PrefValue struct {
	Name    string
	Display string
	Active  bool
}

type Preference struct {
	Name   string
	Values []PrefValue
	Title  string
	Type   string
}

func DefaultPrefMap() map[string]*Preference {
	return map[string]*Preference{
		"written": &Preference{
			Name:   "written",
			Values: []PrefValue{{"true", "True", false}, {"false", "False", true}},
			Title:  "Written Mode",
			Type:   PrefTypeSingle,
		},
	}
}

func (p *Preference) SetVal(val string) {
	//log.Println("SetVal")
	for i, v := range p.Values {
		//log.Println(v.Name, "==", val)
		if v.Name == val {
			//log.Println("true")
			p.Values[i].Active = true
		} else {
			//log.Println("false")
			p.Values[i].Active = false
		}
	}
}

func (p *Preference) GetVal() string {
	//log.Println("GetVal")
	for _, v := range p.Values {
		//log.Println(v.Name, "is Active?")
		if v.Active {
			//log.Println("True!")
			return v.Name
		}
	}
	//log.Println("No match found")
	return ""
}

type User struct {
	Passphrase  string
	Cur         int
	Cards       []UserCard
	Session     string
	Preferences map[string]*Preference
}

func CreateUser(p string, c []UserCard, s string) *User {
	return &User{Passphrase: p, Cards: c, Session: s, Preferences: DefaultPrefMap()}
}

func LoadUser(passphrase string) (*User, error) {
	r, err := os.Open(UserSavePath + passphrase + ".gob")
	if err != nil {
		return nil, err
	}
	defer r.Close()
	u := new(User)
	d := gob.NewDecoder(r)
	err = d.Decode(u)
	if err != nil {
		return nil, err
	}
	if u.Preferences == nil {
		u.Preferences = DefaultPrefMap()
	}
	return u, nil
}

func (u *User) Save() error {
	w, err := os.Create(UserSavePath + u.Passphrase + ".gob")
	if err != nil {
		return err
	}
	defer w.Close()
	e := gob.NewEncoder(w)
	err = e.Encode(u)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) SortCards() {
	sort.Sort(ByPoints(u.Cards))
}

func (u *User) AddCard(m MasterCard) {
	u.Cards = append(u.Cards, UserCard{Master: m})
}

func (u *User) MaybeAddCard(m MasterCard) bool {
	for _, v := range u.Cards {
		if v.Master.Id == m.Id {
			return false
		}
	}
	u.AddCard(m)
	return true
}

func (u *User) SortAndAddCards(master []MasterCard) {
	var addCards bool = false
	u.SortCards()
	for i := 0; i < Top; i++ {
		if !addCards && u.Cards[i].Points() < Benchmark {
			addCards = true
		}
		if addCards {
			for i, v := range master {
				if u.MaybeAddCard(v) {
					break
				} else if i == len(master)-1 {
					log.Println("No cards to add for " + u.Passphrase)
				}
			}
		}
	}
	if addCards {
		u.SortCards()
	}
}
