package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	//"gonotes/service/template"
)

//empty json file need to have [] else 'unexpected end of JSON input'

var dataTable string = "urls.json" //the json file to store with default as original urls.json

var tables = []string{"urls.json", "product.json"}

var tplDir string = "./templates"

//ViewData is a collection of data for the view
type ViewData struct {
	// Page
	// Agent
	Feeds []Feed
	// Rows    [][]string
	//Records      []Record
	//Agents       []Agent
	//Sessions     []Session
	RowCount            int
	Message             string
	PageTitle           string
	ResponseTitle       string
	ResponseBody        string
	ResponseDescription string
	HasSessionID        bool

	URL string

	DataTableDisplay string //none,block for div display of DataTable
}

type Feed struct {
	Title string
	URL   string

	Description  string
	Note         string
	CategoryPath string //Golang/CheatSheet/...
	Tags         []string

	RID         int //RecordID:LastID
	CreatedDate time.Time
	CreatedBy   string //owner of the collection
}

func NewFeed(title, url string) *Feed {
	return &Feed{Title: title, URL: url}
}

type Note struct {
	Title string
	URL   string
	Note  string
}

func NewNote(title, url, note string) *Note {
	return &Note{Title: title, URL: url, Note: note}
}

var SampleFeeds = []Feed{
	{RID: 0, Title: "Create Go service the easy way", URL: "https://medium.com/swlh/create-go-service-the-easy-way-de827d7f07cf"},
	{RID: 1, Title: "Centrifuge – real-time messaging with Go", URL: "https://centrifugal.github.io/centrifugo/blog/intro_centrifuge/"},
	{RID: 2, Title: "Scaling WebSocket in Go and beyond", URL: "https://centrifugal.github.io/centrifugo/blog/scaling_websocket/"},
}

func init() {
	// Trust the augmented cert pool in our client
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	tr := &http.Transport{TLSClientConfig: config}
	client = &http.Client{Transport: tr}

}

func main() {

	ServeRoutes()
}

const (
	IP   = "0.0.0.0"
	PORT = "33"
)

func ServeRoutes() {

	//User Funcs
	http.HandleFunc("/", home)
	http.HandleFunc("/view", view) //
	http.HandleFunc("/add", Add)   //

	fmt.Println("GoNotes listening at", IP, ":", PORT)
	log.Fatal(http.ListenAndServe(IP+":"+PORT, nil))

}

//handles home func, landing page
func home(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "hello")
	//load
	//tmpl := template.LoadTemplate(tplDir, "home.gohtml")
	//run

	//View
	//myUser, hasSessionID := getSessionInfo(r)
	//viewData := &ViewData{PageTitle: "HOME", Message: fmt.Sprint(truncsortwords), Agent: myUser, HasSessionID: hasSessionID}
	//tmpl.Execute(w, viewData)
}

//Add notes
func Add(w http.ResponseWriter, r *http.Request) {
	tmpl := LoadTemplate(tplDir, "add.gohtml")
	if r.Method == http.MethodPost {
		//fmt.Println(r.Form)
		submitVal := r.FormValue("submit")
		dataTable = r.FormValue("DataTable")
		fmt.Println("dataTable:", dataTable)
		fmt.Println(submitVal, r.FormValue("ResponseTitle"), r.FormValue("DataTable"))
		meta := &HTMLMeta{}
		respBody := ""
		//FILTER

		if strings.ToUpper(strings.TrimSpace((submitVal))) == "ENTER" { //from search box
			items := r.FormValue("items")
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace((items))), "http") { //return if not url
				http.Redirect(w, r, "/add", 301)
				return
			}
			bytes := Fetch("GET", items, "")
			//fmt.Println(items, string(bytes))
			respBody = string(bytes)
			meta = extract(strings.NewReader(respBody))

			//json.NewEncoder(os.Stdout).Encode(meta)

			viewData := &ViewData{PageTitle: "ADD", ResponseTitle: meta.Title, ResponseDescription: meta.Description, URL: items, DataTableDisplay: "block"}
			tmpl.Execute(w, viewData)
			return
		}
		if strings.ToUpper(strings.TrimSpace((submitVal))) == "ADD" { //Add button
			items := r.FormValue("items")
			if len(strings.TrimSpace(r.FormValue("ResponseTitle"))) == 0 { //return if empty, prevent adding empty items
				fmt.Println("Invalid ADD, returning as entry is empty...")
				http.Redirect(w, r, "/add", 301)
				return
			}
			//fmt.Println("ADD", items)
			//fmt.Println(submitVal, r.FormValue("ResponseTitle"))
			//meta := extract(strings.NewReader(string(bytes)))

			//json.NewEncoder(os.Stdout).Encode(meta)
			dataRows := LoadData(dataTable)
			dataRows = append(dataRows, Feed{RID: GetNextRID(), Title: r.FormValue("ResponseTitle"), Description: r.FormValue("ResponseDescription"), URL: items})
			SaveFeeds(&dataRows, dataTable)
			viewData := &ViewData{PageTitle: "ADD", DataTableDisplay: "none"}
			tmpl.Execute(w, viewData)
			return
		}
	}
	if r.Method == http.MethodGet {
		viewData := &ViewData{PageTitle: "ADD", DataTableDisplay: "none"}
		tmpl.Execute(w, viewData)
		return
	}

}

//Add notes
func view(w http.ResponseWriter, r *http.Request) {
	tmpl := LoadTemplate(tplDir, "view.gohtml")
	if len(r.FormValue("DataTable")) > 0 { //use default if null,such as onStart
		dataTable = r.FormValue("DataTable")
		fmt.Println("dataTable:", dataTable)
	}

	submitVal := r.FormValue("submit")

	dataRows := LoadData(dataTable)
	var filteredRows = []Feed{}
	r.ParseForm()
	//submitVal := r.FormValue("submit")
	RIDVal := r.FormValue("RecordID")
	fmt.Println("Method", r.Method)
	fmt.Println("dataTable:", dataTable)
	//for k, v := range r.Form {
	//	fmt.Printf("%s = %s\n", k, v)
	//}
	//fmt.Println("PostForm\n", r.PostForm)
	//fmt.Println("Form\n", r.Form)
	fmt.Println(submitVal, RIDVal)
	if r.Method == http.MethodPost {
		//FILTER
		if strings.ToUpper(strings.TrimSpace((submitVal))) == "ENTER" { //from search box
			items := r.FormValue("items")
			for i := len(dataRows) - 1; i >= 0; i-- { //latest first
				match, _ := regexp.MatchString("(?i)"+items, dataRows[i].Title) //matching
				if match {
					filteredRows = append(filteredRows, Feed{RID: dataRows[i].RID, Title: dataRows[i].Title, URL: dataRows[i].URL}) //append
				}
			}
			//Sorted View by RID
			sort.Slice(filteredRows, func(i, j int) bool {
				return filteredRows[i].RID > filteredRows[j].RID
			})
			viewData := &ViewData{PageTitle: "FILTERED", Feeds: filteredRows}
			tmpl.Execute(w, viewData)
			return
		}
		//DELETE
		str := strings.Split(strings.ToUpper(strings.TrimSpace((submitVal))), "_")
		if strings.Contains(str[0], "DEL") {
			for i := len(dataRows) - 1; i >= 0; i-- { //latest first
				DelRID, _ := strconv.Atoi(str[1])
				if dataRows[i].RID != DelRID {
					filteredRows = append(filteredRows, Feed{RID: dataRows[i].RID, Title: dataRows[i].Title, URL: dataRows[i].URL}) //append
				}
			}
			//SaveFile
			SaveFeeds(&filteredRows, dataTable)
			viewData := &ViewData{PageTitle: "NEWS", Feeds: LoadData(dataTable)} //Reload
			tmpl.Execute(w, viewData)
			return
		}

	}
	if r.Method == http.MethodGet {
		rows := LoadData(dataTable)
		// Sort by RID desc(latest on top), keeping original order or equal elements.
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].RID > rows[j].RID
		})
		viewData := &ViewData{PageTitle: "NOTES", Feeds: rows}
		tmpl.Execute(w, viewData)
		return
	}
	if r.Method == http.MethodDelete {
		//Not Supported in Forms
	}
	if r.Method == http.MethodPut {
		//Not Supported in Forms
	}
}

//Edit notes
func Edit(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {

	}
}

//Delete notes
func Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {

	}
}

//AsJSON writes out the header and body for a json payload.
func AsJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

//LoadTemplate loads the template tmplName from tplDir
func LoadTemplate(tplDir, tmplName string) *template.Template {
	t, err := template.New(tmplName).ParseFiles(tplDir + "/" + tmplName)
	if err != nil {
		panic(err)
	}
	return t
}

func LoadData(dbName string) []Feed {
	//return SampleFeeds
	return NewFeedsFromJsonFile(dbName)
}

//Overwrites
func SaveData(data string) {
	WriteFile("urls.json", data)
}

func SaveFeeds(feeds *[]Feed, dbName string) {
	bytes, _ := json.Marshal(feeds)
	WriteFile(dbName, string(bytes))
}

func NewFeedsFromJsonFile(fname string) []Feed {
	file, _ := ioutil.ReadFile(fname)
	var Feeds = []Feed{}
	if err := json.Unmarshal([]byte(file), &Feeds); err != nil {
		log.Fatal(err)
	}
	//fmt.Println(Feeds)
	return Feeds
}

// func SaveReplaceFeedsToJsonFile(fname, data string) {
// 	WriteFile(fname, data)
// }

//WriteFile writes the data as filename, , with create or truncate
func WriteFile(filename string, data string) {
	err := ioutil.WriteFile(filename, []byte(data), 0666)
	if err != nil {
		log.Fatal("cannot read " + filename)
		//return false, err
	}
}

func DumpRequest(w http.ResponseWriter, r *http.Request) {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%q", dump)
}

//Global single instance
var client *http.Client

// -------------------------------------

// Custom user agent.
const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/53.0.2785.143 " +
		"Safari/537.36"
)

// -------------------------------------

func Fetch(method, URL, json string) []byte {
	url, err := url.Parse(URL)
	if err != nil {
		panic(err)
	}
	fmt.Println("Fetch", url.String())
	req, _ := http.NewRequest(method, url.String(), bytes.NewBuffer([]byte(json)))
	//req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("user-agent", userAgent)

	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(bufio.NewReader(resp.Body))

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
	//fmt.Println("response Body:", string(bytes))

	return (bytes)
}

func GetNextRID() int {
	return GetMaxRID() + 1
}

func GetMaxRID() int {
	maxRID := 0
	data := LoadData(dataTable)
	for _, v := range data {
		if v.RID > maxRID {
			maxRID = v.RID
		}
	}
	return maxRID
}
