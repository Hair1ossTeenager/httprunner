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
  headers:
    Hrp-Test: ${GetRandomString()}
    content-type: application/json;charset=UTF-8
    authorization: Bearer ${authToken}
    tntid: ${tntId}
teststeps:
  - name: "%s api test"
    request:
      method: %s
      url: %s
    %s:
      %s
    validate:
      - eq: ["status_code", 200]
      - eq: ["body.type", success]
`

var reqParamKey = map[string]string{
	"get":    "params",
	"put":    "body",
	"post":   "body",
	"delete": "params",
}

var convertSwaggerCmd = &cobra.Command{
	Use:          "convertSwagger $url $path",
	Short:        "convert swagger doc.json format to HttpRunner YAML cases",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: false,
	PreRun: func(cmd *cobra.Command, args []string) {
		setLogLevel(logLevel)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[1]
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
				if !ok {
					continue
				}
				var ps []string
				for _, param := range params {
					p, ok := param.(map[string]interface{})
					if !ok {
						continue
					}
					pName, ok := p["name"].(string)
					if !ok {
						continue
					}
					if pName == "body" {
						schema := p["schema"].(map[string]interface{})
						sKey := schema["$ref"].(string)
						dKey := strings.ReplaceAll(sKey, "#/definitions/", "")
						definition, ok := definitions[dKey].(map[string]interface{})
						if ok {
							properties, ok := definition["properties"].(map[string]interface{})
							if ok {
								var pList []string
								for k, _ := range properties {
									pList = append(pList, fmt.Sprintf("%s: \"\"", k))
								}
								pName = strings.Join(pList, "\n      ")
							}
						}
						ps = append(ps, pName)
					} else {
						ps = append(ps, fmt.Sprintf("%s: \"\"", pName))
					}
				}
				yamlContent := fmt.Sprintf(yamlTemplate, urlPath, urlPath, strings.ToUpper(method), urlPath, reqParamKey[method], strings.Join(ps, "\n      "))
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
