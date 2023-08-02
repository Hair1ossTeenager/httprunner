package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/httprunner/httprunner/v4/hrp/internal/json"
	"github.com/httprunner/httprunner/v4/hrp/pkg/tool"
)

const yamlTemplate = `config:
  name: %s api test
teststeps:
  - name: "%s api test"
    request:
      method: %s
      url: %s
      %s
    validate:
      - eq: ["status_code", 200]
      - eq: ["body.type", success]
`

const paramTemplate = `%s:
        %s
`

const (
	arraySep   = "\n          "
	defaultSep = "\n        "
	paramSep   = "\n      "
)

var convertSwaggerCmd = &cobra.Command{
	Use:          "convertSwagger $url $path",
	Short:        "convert swagger doc.json format to HttpRunner YAML cases",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: false,
	PreRun: func(cmd *cobra.Command, args []string) {
		setLogLevel(logLevel)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		path := ""
		if len(args) > 1 {
			path = args[1]
		}
		u, err := url.Parse(args[0])
		if err != nil {
			return err
		}
		domain := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		res, err := tool.DoGet(domain, u.Path, nil, nil)
		if err != nil {
			return err
		}
		doc := make(map[string]interface{})
		if err := json.Unmarshal(res, &doc); err != nil {
			return err
		}
		definitions := doc["definitions"].(map[string]interface{})
		for urlPath, obj := range doc["paths"].(map[string]interface{}) {
			for method, o := range obj.(map[string]interface{}) {
				z := o.(map[string]interface{})
				params, ok := z["parameters"].([]interface{})
				var queries []string
				var bodies []string
				var formData []string
				if ok {
					for _, param := range params {
						p, ok := param.(map[string]interface{})
						if !ok {
							fmt.Println(urlPath)
							continue
						}
						in := p["in"].(string)
						pName, ok := p["name"].(string)
						if !ok {
							fmt.Println(urlPath)
							continue
						}

						if in == "body" {
							schema, ok := p["schema"].(map[string]interface{})
							if !ok {
								fmt.Println(urlPath)
								continue
							}
							sKey := ""
							isArray := false
							if _, ok := schema["type"]; ok {
								items := schema["items"].(map[string]interface{})
								sKey = items["$ref"].(string)
								isArray = true
							} else {
								sKey, ok = schema["$ref"].(string)
								if !ok {
									continue
								}
							}
							dKey := strings.ReplaceAll(sKey, "#/definitions/", "")
							definition, ok := definitions[dKey].(map[string]interface{})
							if ok {
								properties, ok := definition["properties"].(map[string]interface{})
								if ok {
									var pList []string
									for k, _ := range properties {
										pList = append(pList, fmt.Sprintf("%s: \"\"", k))
									}
									sep := defaultSep
									if isArray {
										pList[0] = fmt.Sprintf("- %s", pList[0])
										sep = arraySep
									}
									pName = strings.Join(pList, sep)
								}
							}
							bodies = append(bodies, pName)
						} else if in == "formData" {
							formData = append(formData, fmt.Sprintf("%s: \"\"", pName))
						} else if in == "query" {
							queries = append(queries, fmt.Sprintf("%s: \"\"", pName))
						} else {
							continue
						}
					}
				}
				param := ""
				queryParam := ""
				bodyParam := ""
				uploadParam := ""
				if len(queries) > 0 {
					queryParam = fmt.Sprintf(paramTemplate, "params", strings.Join(queries, defaultSep))
				}
				if len(bodies) > 0 {
					bodyParam = fmt.Sprintf(paramTemplate, "body", strings.Join(bodies, defaultSep))
				}
				if len(formData) > 0 {
					uploadParam = fmt.Sprintf(paramTemplate, "upload", strings.Join(formData, defaultSep))
				}
				for _, v := range []string{queryParam, bodyParam, uploadParam} {
					if v != "" {
						v = strings.Trim(v, defaultSep)
						param = param + v + paramSep
					}
				}
				param = strings.Trim(param, paramSep)
				param = strings.Trim(param, " ")
				yamlContent := fmt.Sprintf(yamlTemplate, urlPath, urlPath, strings.ToUpper(method), urlPath, param)
				if path == "" {
					fmt.Println(yamlContent)
					continue
				}
				fileName := strings.TrimLeft(strings.ReplaceAll(urlPath, "/", "-"), "-")
				filePath := fmt.Sprintf("%s/%s.yaml", path, fileName)
				if _, err := writeFile(filePath, yamlContent); err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(convertSwaggerCmd)
}

func writeFile(path, content string) (int, error) {
	dir := filepath.Dir(path)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			fmt.Println(err)
		}
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return file.WriteString(content)
}
