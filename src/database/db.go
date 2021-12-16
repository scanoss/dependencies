package dbdependencies

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/package-url/packageurl-go"
)

type Dependency struct {
	gorm.Model
	Purl                 string
	Source               int64
	Dependency_vendor    string
	Dependency_component string
	Dependency_version   string
}
type DependencyItem struct {
	gorm.Model
	Purl_hash         string
	Purl_version_hash string
	Dep_vendor        string
	Dep_component     string
	Dep_version       string
}
type ShortDependencyItem struct {
	gorm.Model
	Purl_hash     string
	Dep_vendor    string
	Dep_component string
	Dep_version   string
}

type Project struct {
	gorm.Model
	Mine_id   int16
	Purl_type string
	Vendor    string
	Component string
	License   string
	Versions  int16
	Purl_name string
}

type ProjectInfo struct {
	Source       string
	Vendor       string
	Component    string
	License      string
	Versions     int16
	Dependencies []ProjectInfo
}
type TMines struct {
	//gorm.Model
	Id       int16
	Name     string
	PurlType string
}

var db *gorm.DB
var err error
var dbHost string
var dbPort int
var dbName string
var dbUsr string
var dbPsw string

var dependencies []Dependency
var handlers [100]func(purl string) []DependencyItem

var cacheMines map[string]TMines

func cacheMinesTable() {

	var queryResult []TMines
	cacheMines = make(map[string]TMines)
	db.Raw("select * from mines m").Scan(&queryResult)
	for i := 0; i < len(queryResult); i++ {
		tmpMine := queryResult[i]
		cacheMines[queryResult[i].Name] = tmpMine
		//fmt.Printf("%d %s %s\n", queryResult[i].Id, queryResult[i].Name, queryResult[i].PurlType)
	}
	//	fmt.Println(cacheMines)

}

func getMineId(purlType string) int16 {
	for _, element := range cacheMines {
		if element.PurlType == purlType {
			return element.Id
		}
	}
	return -1

}
func OpenDB() {
	//router := mux.NewRouter()
	handlers[0] = HandleMavenDep

	handlers[1] = HandleRubyDep
	handlers[2] = HandleNpmDep

	var connectStr string
	connectStr = fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable password=%s", dbHost, dbPort, dbUsr, dbName, dbPsw)
	//	db, err = gorm.Open("postgres", "host=ns3163276.ip-51-91-82.eu port=5432 user=metaquery dbname=components sslmode=disable password=dKA5tLM5y6eb")
	//fmt.Println(connectStr)
	db, err = gorm.Open("postgres", connectStr)
	if err != nil {
		panic("Failed to connect database")
	}

	//defer db.Close()

	db.AutoMigrate(&Dependency{})
	cacheMinesTable()
	//router.HandleFunc("/dependency", GetDependency).Methods("GET")

	//	handler := cors.Default().Handler(router)

	//log.Fatal(http.ListenAndServe("0.0.0.0:8080", handler))
}

func CloseDB() {
	db.Close()
}
func GetDependency(w http.ResponseWriter, r *http.Request) {

	purl, ok := r.URL.Query()["purl"]
	if !ok {
		return
	}

	var prj []Project
	prj = GetProjectInfo(purl[0], 5)
	json.NewEncoder(w).Encode(&prj)

}

func HandleMavenDep(purl string) []DependencyItem {
	///var di []ShortDependencyItem

	/*purl_hash := md5.Sum([]byte(purl))
	str_purl_hash := fmt.Sprintf("%x", purl_hash)
	str_purl_hash = strings.ToUpper(str_purl_hash)
	//	log.Printf("query %s %s", str_purl_hash)
	//	var depsVendor []string
	//	db.Raw("select purl_hash, dep_vendor, dep_component, dep_version from maven_dependencies md where upper(md.purl_hash) = ?", str_purl_hash).Scan(&di)

	//	db.Raw("select distinct * from maven_dependencies md where upper(md.purl_hash) = ?", str_purl_hash).Scan(&di)
	//log.Print(di)
	/*for i := 0; i < len(di); i++ {
		fmt.Printf("\n{\n component: %s \n version: %s\n},", di[i].Dep_component, di[i].Dep_version)

	}
	//	fmt.Printf("[%s] has %d dependencies", purl, len(di))
	return di*/
	fmt.Println("handle MAVEN")
	return nil

}

func HandleNpmDep(purl string) []DependencyItem {
	fmt.Println("handle NPM")
	return nil
}

func HandleRubyDep(purl string) []DependencyItem {
	fmt.Println("handle RUBY")
	return nil
}

func InitDB(host string, port int, name string, usr string, psw string) {
	dbHost = host
	dbName = name
	dbPort = port
	dbUsr = usr
	dbPsw = psw
}

/**
@see try to implement as a variadic function
*/

func GetProjectInfo(purl string, depth int) []Project {

	var prj []Project
	instance, _ := packageurl.FromString(purl)
	purlname := ""
	purltype := ""

	if instance.Namespace != "" {
		purlname = instance.Namespace + "/" + instance.Name
	} else {
		purlname = instance.Name
	}
	purltype = instance.Type

	/*db.Raw("select p.Mine_id,	p.Vendor,	p.Component,	p.License, p.Versions, p.purl_name "+
	"from projects p, Mines m "+
	"where m.purl_type = ? and m.id = p.mine_id and p.purl_name = ?", purltype, purlname).Scan(&prj)
	*/

	db.Raw("select a.Mine_id,	a.Vendor,	a.Component,	a.License, a.Version, a.purl_name "+
		"from all_urls a "+
		"where a.mine_id = ? and a.purl_name = ?", getMineId(purltype), purlname).Scan(&prj)

	for k := 0; k < len(prj); k++ {
		var prjExtra []Project
		//prj[k].License = ""
		if prj[k].License == "" {
			log.Println("Searching on Projects table")
			db.Raw("select p.Mine_id,	p.Vendor,	p.Component,	p.License, p.Versions, p.purl_name "+
				"from projects p "+
				"where p.mine_id = ? and p.purl_name = ?", getMineId(purltype), purlname).Scan(&prjExtra)
			if len(prjExtra) > 0 {
				prj[k].License = prjExtra[0].License
			}

		}

	}

	//log.Printf("Got %d results", len(prj))
	return prj

}
