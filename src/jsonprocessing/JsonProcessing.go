package jsonprocessing

import (
	"encoding/json"
	"fmt"
	"log"

	dbdependencies "scanoss.com/dependencies/src/database"
)

type JsonLicense struct {
	Name string `json:"name"`
}
type JsonDependency struct {
	Purl      string        `json:"purl"`
	Component string        `json:"component"`
	Vendor    string        `json:"vendor"`
	Version   int16         `json:"version"`
	License   []JsonLicense `json:"license"`
}

type JsonKey struct {
	Id          string           `json:"id"`
	Status      string           `json:"status"`
	Dependecies []JsonDependency `json:"dependencies"`
}

var jsonResponse map[string]JsonKey

type item_process func(string) interface{}

func parseJson(jsonText []byte) (interface{}, error) {
	var parsed interface{}
	err := json.Unmarshal(jsonText, &parsed)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func json_process(parsed interface{}, f item_process, output *[]JsonDependency) {

	switch parsed.(type) {
	case map[string]interface{}:
		table := parsed.(map[string]interface{})
		for k := range table {
			if k == "purl" {
				//	fmt.Printf("\n%s\n", table[k].(string))

				result := f(table[k].(string))
				//log.Println(result)
				var r JsonDependency
				p := result.([]dbdependencies.Project)
				if len(p) > 0 {
					r.Vendor = p[0].Vendor
					lic := JsonLicense{Name: p[0].License}
					r.License = append(r.License, lic)
					r.Purl = p[0].Purl_name
					r.Component = p[0].Component
					r.Version = p[0].Versions
					//fmt.Println(r)
					*output = append(*output, r)
				}
			} else {
				//		log.Printf("%v-------->", k)
				json_process(table[k], f, output)
			}
		}
	//	log.Printf("id:%s name:%s\n", table["purls"], table["name"])
	case []interface{}:
		list := parsed.([]interface{})
		//		log.Printf("list of %d items\n", len(list))
		for i := range list {
			//		log.Println(list[i])
			json_process(list[i], f, output)
		}
	case interface{}:
		/*single := parsed.(string)
		//		log.Print("Item: ")
		log.Println(single)
		/*
			result := f(single)
			var r JsonDependency
			p := result.([]QueryDeps.Project)
			r.Vendor = p[0].Vendor
			lic := JsonLicense{Name: p[0].License}
			r.License = append(r.License, lic)
			r.Purl = p[0].Purl_name
			r.Component = p[0].Component
			//r.Version = p[0].Versions
			//fmt.Println(r)
			*output = append(*output, r)*/
	default:
		panic(fmt.Errorf("type %T unexpected", parsed))
	}
}

func query(purl string) interface{} {
	return dbdependencies.GetProjectInfo(purl, 0)
}

func DepsProcess(input string) string {
	data := []byte(input)
	//dbdependencies.OpenDB()

	parsed, err := parseJson(data)
	if err != nil {
		panic(err) // malformed input
	}

	jsonResponse = make(map[string]JsonKey)

	table := parsed.(map[string]interface{})
	for k := range table {
		log.Printf("%s->", k)
		key_results := JsonKey{Id: "dependency", Status: "pending"}
		json_process(table[k], query, &key_results.Dependecies)
		jsonResponse[k] = key_results
	}

	b, err := json.MarshalIndent(jsonResponse, "", "  ")

	if err != nil {

		fmt.Println("error:", err)

	}

	//	os.Stdout.Write(b)
	//dbdependencies.CloseDB()

	return string(b)
}
