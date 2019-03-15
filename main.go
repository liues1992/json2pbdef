package main

import (
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .ArgsUsage}}{{.ArgsUsage}}{{else}} jsonData{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
`
	app.Name = "json2protodef"
	app.Usage = "Create protobuf definition from json data"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "custom message name",
		},
		cli.StringFlag{
			Name:  "package",
			Usage: "add package name",
		},
		cli.BoolFlag{
			Name:  "header",
			Usage: "with file header",
		},
	}
	app.Action = func(c *cli.Context) (err error) {
		var hasStdIn = false
		if c.NArg() == 0 {
			stat, _ := os.Stdin.Stat()
			if stat.Size() == 0 {
				return cli.ShowAppHelp(c)
			}
			hasStdIn = true
		}
		var data []byte
		if hasStdIn {
			data, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return
			}
		} else {
			arg0 := c.Args().Get(0)
			data = []byte(arg0)
			if data[0] != '{' {
				// not json data
				if len(arg0) >= 7 && (arg0[:5] == "http:" || arg0[:6] == "https:") {
					data, err = getHttpContent(arg0)
					if err != nil {
						return
					}
				} else {
					data, err = ioutil.ReadFile(arg0)
					if err != nil {
						return
					}
				}
			}

		}
		j, err := simplejson.NewJson(data)
		if err != nil {
			return errors.WithMessage(err, "Invalid json")
		}
		m, err := j.Map()
		if err != nil {
			return errors.WithMessage(err, "json root must be a object")
		}

		var customName = c.String("name")
		if customName == "" {
			customName = "Message"
		}
		output, err := messageFromJsonObject(customName, m, 0, []string{})
		if err != nil {
			return
		}
		if c.Bool("header") {
			fmt.Println("syntax = \"proto3\";\n")
		}
		if p := c.String("package"); p != "" {
			fmt.Println("package " + p + ";\n")
		}
		fmt.Println(strings.Join(output, "\n"))
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func isValidFieldName(name string) bool {
	var a = "^[a-zA-Z_][a-zA-Z_0-9]*$"
	var r, _ = regexp.Compile(a)
	return r.Match([]byte(name))
}

func getKeyPath(keyPaths []string, key string) string {
	r := ""
	for _, v := range keyPaths {
		r = r + "." + v
	}
	return r + "." + key
}

// @param name message name
// @param m a map from json
// @return protobuf message definition
//		   code in string slice
func messageFromJsonObject(name string, obj map[string]interface{}, indent int, keyPaths []string) (ret []string, err error) {
	name = strcase.ToCamel(name)
	if len(obj) == 0 {
		err = errors.New("Cannot infer structure from empty object, keypath=" + getKeyPath(keyPaths, ""))
		return
	}
	i := 1
	ret = append(ret, strings.Repeat(" ", indent)+"message "+name+" {")
	//sort the keys to get consistent output
	var keys []string
	for k, _ := range obj {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, k := range keys {
		if !isValidFieldName(k) {
			err = errors.New("Invalid field name: \"" + getKeyPath(keyPaths, k) + "\"")
			return
		}
		v := obj[k]
		if v == nil {
			err = errors.New("Cannot infer structure from null value, keypath=" + getKeyPath(keyPaths, k))
			return
		}
		var isRepeated = false
		var isMap = false
		switch reflect.TypeOf(v).Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(v)
			if s.Len() == 0 {
				err = errors.New("Connot infer a empty array, keypath=" + getKeyPath(keyPaths, k))
				return
			}
			isRepeated = true
			v = s.Index(0).Interface()
		}
		var typ string
		var outputObject map[string]interface{}
		typ, isMap, outputObject, err = getType(v, strcase.ToCamel(k), true, getKeyPath(keyPaths, k))
		if err != nil {
			return
		}
		if outputObject != nil {
			var lines []string
			var subKeyPaths = keyPaths
			subKeyPaths = append(subKeyPaths, k)
			if isRepeated {
				subKeyPaths = append(subKeyPaths, "0")
			}
			lines, err = messageFromJsonObject(k, outputObject, indent+4, subKeyPaths)
			if err != nil {
				return
			}
			ret = append(ret, lines...)
		}
		var field string
		if isMap {
			field = fmt.Sprintf("map<int64, %s> %s = %d;", typ, k, i)
		} else if isRepeated {
			field = fmt.Sprintf("repeated %s %s = %d;", typ, k, i)
		} else {
			field = fmt.Sprintf("%s %s = %d;", typ, k, i)
		}
		i += 1
		ret = append(ret, strings.Repeat(" ", indent+4)+field)
	}
	ret = append(ret, strings.Repeat(" ", indent)+"}")
	return
}

func getType(v interface{}, name string, allowMap bool, keyPath string) (typ string, isMap bool, outputObject map[string]interface{}, err error) {
	switch v.(type) {
	case json.Number:
		num := v.(json.Number)
		_, e := num.Int64()
		if e == nil {
			typ = "int64"
		} else {
			typ = "float64"
		}
	case string:
		typ = "string"
	case bool:
		typ = "bool"
	case map[string]interface{}:
		// this obj may be "map"
		// which means variable keys map to same value structure
		// key must be number
		// eg.
		// {
		//    "1": "a",
		//    "2": "b"
		// }
		item := v.(map[string]interface{})
		if len(item) == 0 {
			err = errors.New("Cannot infer a empty map, keypath=" + keyPath)
			return
		}
		var firstVal interface{}
		var firstKey string
		for k1, v1 := range item {
			firstVal = v1
			firstKey = k1
			_, e := strconv.ParseInt(k1, 10, 64)
			if e == nil {
				isMap = true
			}
			break
		}
		if !allowMap && isMap {
			err = errors.New("Nested map is not allowed, keypath=" + keyPath)
			return
		}
		if isMap {
			typ, _, outputObject, err = getType(firstVal, name, false, keyPath+"."+firstKey)
			if err != nil {
				return
			}
		} else {
			outputObject = item
			typ = name
			return
		}
	default:
		err = errors.New(fmt.Sprintf("unknow type %T", v))
	}
	return
}

func getHttpContent(url string) (ret []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	ret = body
	return
}
