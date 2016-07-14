package main

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const Benchmark = 15
const MasterFile = "./master.gob"
const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var quit = make(chan int)
var Top int = 8
var mCards []MasterCard
var UserList map[string]*User = make(map[string]*User, 0)
var ErrNotLoggedIn error = errors.New("User not logged in.")

func logError(e error, req *http.Request) {
	log.Println(time.Now(), " ", req.RemoteAddr, ":", e)
}

func getUser(req *http.Request) (*User, error) {
	s, err := req.Cookie("session")
	if err != nil {
		// Session Not Found
		if err == http.ErrNoCookie {
			log.Println("getUser-noSession:", s)
			return nil, ErrNotLoggedIn
		}
		// Unhandled Error
		log.Println("getUser-unhandled:", err)
		return nil, err
	}
	if UserList[s.Value] == nil {
		return nil, ErrNotLoggedIn
	}
	// Stored Session Found
	return UserList[s.Value], nil
}

func genSessionId() string {
	hash := make([]byte, 16)
	rand.Read(hash)
	for i := range hash {
		hash[i] = charset[int(hash[i])%len(charset)]
	}
	for UserList[string(hash)] != nil {
		hash = make([]byte, 16)
		rand.Read(hash)
	}
	return string(hash)
}

//TODO

func logoutUser(u *User) {
	UserList[u.Session] = nil
	u.Session = ""
	if err := u.Save(); err != nil {
		log.Println(time.Now(), " Logout Error:", err)
	}
}

/*func bootstrapFlash() (flashArr []*flashcard) {
	flashMap := make(map[string]flashcard)
	file, err := os.Open("./dianhuabookmarks.csv")
	if err != nil {
		log.Println("boostrapDict: ", err)
	}
	defer file.Close()
	r := csv.NewReader(file)
	rec, err := r.Read()
	for err == nil {
		flashMap[rec[0]] = flashcard{
			Simplified:  rec[0],
			Traditional: rec[1],
			Pinyin:      rec[2],
			English:     strings.Split(rec[3][1:len(rec[3])-1], "/"),
			LastSeen:    time.Now(),
			Correct:     0,
			Wrong:       0}
		rec, err = r.Read()
	}
	flashArr = make([]*flashcard, 0, len(flashMap))
	for _, v := range flashMap {
		flashArr = append(flashArr, &flashcard{
			Simplified:  v.Simplified,
			Traditional: v.Traditional,
			Pinyin:      v.Pinyin,
			English:     v.English,
			LastSeen:    v.LastSeen,
			Correct:     v.Correct,
			Wrong:       v.Wrong})
	}
	log.Println("bootstrapFlash:", err)
	return
}*/

func bootstrap() (flashArr []MasterCard) {
	file, err := os.Open("./GLOSS.txt")
	if err != nil {
		fmt.Println(err)
		log.Fatal("boostrapDict: ", err)
	}
	defer file.Close()
	r := csv.NewReader(file)
	r.Comma = ';'
	rec, err := r.Read()
	i := 0
	order := make([]int, 4)
	for i := range rec {
		if rec[i] == "english" {
			order[0] = i
		} else if rec[i] == "simplified" {
			order[1] = i
		} else if rec[i] == "traditional" {
			order[2] = i
		} else if rec[i] == "pinyin" {
			order[3] = i
		}
	}
	rec, err = r.Read()
	flashArr = make([]MasterCard, 0)
	for err == nil {
		i++
		insert := false
		insertPoint := 0
		for _, v := range flashArr {
			if v.Simplified == rec[order[1]] {
				insert = true
				break
			}
			insertPoint++
		}
		if !insert {
			flashArr = append(flashArr, MasterCard{
				Id:          i,
				Simplified:  rec[order[1]],
				Traditional: rec[order[2]],
				Pinyin:      rec[order[3]],
				English:     strings.Split(rec[order[0]], "/"),
			})
		} else {
			flashArr[insertPoint].English = append(flashArr[insertPoint].English, strings.Split(rec[order[0]][1:len(rec[order[0]])-1], "/")...)
			i--
		}
		rec, err = r.Read()
	}
	fmt.Println(err)

	// Deduplicate definitions
	for i, _ := range flashArr {
		if len(flashArr[i].English) > 1 {
			strDedup := make([]string, 0)
			for _, s := range flashArr[i].English {
				seen := false
				for _, test := range strDedup {
					if s == test {
						seen = true
					}
				}
				if !seen {
					strDedup = append(strDedup, s)
				}
			}
			flashArr[i].English = strDedup
		}
	}
	fmt.Println(i)
	return
}

func loadMaster() ([]MasterCard, error) {
	file, err := os.Open(MasterFile)
	if err != nil {
		if err.Error() == "open ./master.gob: The system cannot find the file specified." {
			mArr := bootstrap()
			saveMaster(&mArr)
			return mArr, nil
		}
		return nil, err
	}
	defer file.Close()
	r := gob.NewDecoder(file)
	mArr := make([]MasterCard, 0, 5000)
	err = r.Decode(&mArr)
	if err != nil {
		return nil, err
	}
	return mArr, nil
}

func saveMaster(m *[]MasterCard) error {
	file, err := os.Create(MasterFile)
	if err != nil {
		return err
	}
	defer file.Close()
	w := gob.NewEncoder(file)
	w.Encode(m)
	return nil
}

func loadFlash(file *os.File) (flashArr []*flashcard) {
	flashArr = make([]*flashcard, 0, 5000)
	r := gob.NewDecoder(file)
	err := r.Decode(&flashArr)
	if err != nil {
		log.Fatal("loadFlash:", err)
	}
	return
}

func sortFlashcards(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(rw, time.Now().String())
	sort.Sort(ById{mCards})
	fmt.Fprintf(rw, time.Now().String())
}

func runServer(addr string, handler http.Handler) {
	err := http.ListenAndServe(addr, handler)
	if err != nil {
		log.Fatal("http error: " + err.Error())
	}
}

func searchMaster(q string) MasterCard {
	for _, v := range mCards {
		if v.Simplified == q {
			return v
		}
	}
	return *new(MasterCard)
}

/*func createOlreich() {
	f, _ := os.Open("./flash.gob")
	defer f.Close()
	a := loadFlash(f)
	c := make([]UserCard, 0)
	for _, v := range a {
		if v.Correct > 0 || v.Wrong > 0 {
			m := searchMaster(v.Simplified)
			c = append(c, UserCard{Master: m, LastSeen: v.LastSeen, Correct: v.Correct, Wrong: v.Wrong})
		}
	}
	u := CreateUser("olreich", c, "")
	u.Save()
}*/

func saveAllUsers() {
	for _, v := range UserList {
		v.Save()
	}
}

/* TODO:
Multiple Sessions
*/
func main() {
	defer saveAllUsers()
	// Setup Logging
	logFile, err := os.Create("./log.txt")
	if err != nil {
		log.Fatal("main:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Setup Flashcards
	mCards, err = loadMaster()
	if err != nil {
		log.Fatalln("main: ", err)
	}

	// HttpHandlers
	http.HandleFunc("/", homeHandler)

	// Static file service
	http.Handle("/static/", http.FileServer(http.Dir("./")))

	http.HandleFunc("/quit", quitHandler)
	go runServer(":8080", nil)
	fmt.Println("Server Ready")
	<-quit
}
