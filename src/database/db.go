/*
 SPDX-License-Identifier: MIT
   Copyright (c) 2021, SCANOSS
   Permission is hereby granted, free of charge, to any person obtaining a copy
   of this software and associated documentation files (the "Software"), to deal
   in the Software without restriction, including without limitation the rights
   to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
   copies of the Software, and to permit persons to whom the Software is
   furnished to do so, subject to the following conditions:
   The above copyright notice and this permission notice shall be included in
   all copies or substantial portions of the Software.
   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
   OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
   THE SOFTWARE.
*/
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
	}

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
	handlers[0] = HandleMavenDep

	handlers[1] = HandleRubyDep
	handlers[2] = HandleNpmDep

	var connectStr string
	connectStr = fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable password=%s", dbHost, dbPort, dbUsr, dbName, dbPsw)
	db, err = gorm.Open("postgres", connectStr)
	if err != nil {
		panic("Failed to connect database")
	}

	db.AutoMigrate(&Dependency{})
	cacheMinesTable()

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

	db.Raw("select a.Mine_id,	a.Vendor,	a.Component,	a.License, a.Version, a.purl_name "+
		"from all_urls a "+
		"where a.mine_id = ? and a.purl_name = ?", getMineId(purltype), purlname).Scan(&prj)

	for k := 0; k < len(prj); k++ {
		var prjExtra []Project
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
