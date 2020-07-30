package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/spf13/viper"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
}

// Defines the struct that holds config info about the Apigee environment
// This is filled in by nr-apigee_metrics-settings.yml.
type configStruct struct {
	ProxyURL  string   `yaml:"proxyURL"`
	TimeRange int      `yaml:"timeRange"`
	Dimension string   `yaml:"dimension"`
	Queries   []string `yaml:"queries"`
	Apigee    struct {
		Orgs []struct {
			Org      string   `yaml:"org"`
			Baseurl  string   `yaml:"baseurl"`
			UserID   string   `yaml:"userID"`
			Password string   `yaml:"password"`
			Envs     []string `yaml:"envs"`
		} `yaml:"orgs"`
	} `yaml:"apigee"`
}

type apigeeJSONstr struct {
	Environments []struct {
		Dimensions []struct {
			Metrics []struct {
				Name   string        `json:"name"`
				Values []json.Number `json:"values"`
			} `json:"metrics"`
			Name string `json:"name"`
		} `json:"dimensions"`
		Name string `json:"name"`
	} `json:"environments"`
	MetaData struct {
		Errors  []interface{} `json:"errors"`
		Notices []string      `json:"notices"`
	} `json:"metaData"`
}

const (
	integrationName    = "com.newrelic.nri-apigee"
	integrationVersion = "0.2.0"
)

var (
	args       argumentList
	configData configStruct
)

func populateMetrics(integration *integration.Integration, apigeeJSON apigeeJSONstr, apigeeOrg string) error {
	log.Debug("Beginning populate metrics.")
	for _, environment := range apigeeJSON.Environments {
		for _, dimension := range environment.Dimensions {
			log.Debug("Processing - Org : " + apigeeOrg + ", Env : " + environment.Name + ", Dim : " + dimension.Name)
			entity := integration.LocalEntity()
			ms := entity.NewMetricSet("ApigeeSample")
			ms.SetMetric("ApigeeOrg", apigeeOrg, metric.ATTRIBUTE)
			ms.SetMetric("ApigeeEnv", environment.Name, metric.ATTRIBUTE)
			ms.SetMetric("ApigeeProxyName", dimension.Name, metric.ATTRIBUTE)

			for _, apigeeMetric := range dimension.Metrics {
				//log.Debug("Metric Name : " + apigeeMetric.Name + " = " + apigeeMetric.Values[0])
				log.Debug("Metric Name : %s = %s", apigeeMetric.Name, apigeeMetric.Values[0])
				ms.SetMetric(apigeeMetric.Name, apigeeMetric.Values[0], metric.GAUGE)
			}
		}
	}
	return nil
}

func readConfig() {
	log.Debug("In readConfig")
	viper.SetConfigName("nri-apigee_metrics-settings")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	err = viper.Unmarshal(&configData)
	if err != nil {
		panic(fmt.Errorf("fatal error config unmarshal: %s", err))
	}
}

// Build the query we're going to send to Apigee's RESTful interface.
func buildApigeeMetricQuery(apigeeBase string, apigeeOrg string, apigeeEnv string, apigeeTotalQuery string, apigeeDimension string, apigeeTimeRange time.Duration) string {
	log.Debug("In buildApigeeMetricQuery")

	//Apigee requires time specificed in UTC. Their docs say that data can take up to 10 minutes to post, so (to be safe)
	// we are looking back 15 minutes and getting 5 minute aggregates.
	// We multiply by -1 because the function is adding, not subtracting.
	var now = time.Now().UTC()
	var timeRangeStart = now.Add(time.Minute * -15)
	var timeRangeEnd = timeRangeStart.Add(time.Minute*-1 + apigeeTimeRange)

	//I'm sure I could do something fancy with the URL module, but for now, I'm doing this.
	var timeRange = strings.Replace(timeRangeStart.Format("01/02/2006 15:04")+"~"+timeRangeEnd.Format("01/02/2006 15:04"), " ", "%20", -1)
	var apigeeURL = apigeeBase + "/" + apigeeOrg + "/environments/" + apigeeEnv + "/stats/" + apigeeDimension + "?" + apigeeTotalQuery + "&timeRange=" + timeRange
	log.Debug("Apigee URL : %s", apigeeURL)

	return apigeeURL
}

//Query Apigee for the list of environments associated with the org.
func buildApigeeEnvQuery(apigeeOrg string, apigeeBase string) string {
	log.Debug("In buildAPigeeEnvQuery")
	log.Debug("Building query for environment: %s", apigeeOrg)
	apigeeEnvURL := apigeeBase + "/" + apigeeOrg + "/environments"
	log.Debug("Environment query URL: %s", apigeeEnvURL)

	return apigeeEnvURL
}

// Send the query we built to Apigee and capture the result.
func executeApigeeQuery(apigeeURL string, apigeeUser string, apigeePass string) []byte {
	log.Debug("In executeApigeeQuery")
	client := &http.Client{}
	req, err := http.NewRequest("GET", apigeeURL, nil)
	req.SetBasicAuth(apigeeUser, apigeePass)
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(resp.Status)
	log.Debug("Response: " + resp.Status)

	page, err := ioutil.ReadAll(resp.Body)
	// fmt.Println("Page: \n")
	// fmt.Printf("%s\n\n", page)

	return page
}

// Process the returned JSON query result into an object
func processApigeeJSON(apigeeResp []byte) apigeeJSONstr {
	log.Debug("In processsApigeeJSON")
	var apigeeJSON apigeeJSONstr
	if err := json.Unmarshal(apigeeResp, &apigeeJSON); err != nil {
		panic(fmt.Errorf("fatal error processing Apigee JSON: %s", err))
	}
	return apigeeJSON
}

func main() {
	//Initialize some variables
	var apigeeMetricURL string
	var apigeeResp []byte
	var apigeeJSON apigeeJSONstr
	var apigeeTotalQuery string
	var apigeeEnvs []string

	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	fatalIfErr(err)
	log.SetupLogging(args.Verbose)

	log.Debug("Starting Up - version: " + integrationVersion)

	//Get the configuration data about the Apigee instance we're going to query.
	readConfig()

	var apigeeDimension = configData.Dimension
	var apigeeTimeRange = time.Duration(configData.TimeRange) * time.Minute
	var ProxyURL = configData.ProxyURL

	log.Debug("Configuration has been read in.")
	log.Debug("Proxy URL: %s", ProxyURL)
	log.Debug("Apigee Dimension: %s", apigeeDimension)
	log.Debug("Apigee Time Range: %s", apigeeTimeRange)

	//If a proxyURL is configured, set the appropriate environment variable
	if ProxyURL != "" {
		log.Debug("Proxy setting detected. Configuring OS environment.")
		os.Setenv("HTTP_PROXY", ProxyURL)
	}

	//We can pass all of our queries at once to Apigee - we'll see how big a mess this makes
	apigeeTotalQuery = "select=" + strings.Join(configData.Queries, ",")
	log.Debug("Apigee Total Query =" + apigeeTotalQuery)

	log.Debug("Ready to poll Apigee")
	if args.All() || args.Metrics {
		for _, apigeeOrg := range configData.Apigee.Orgs {
			apigeeEnvURL := buildApigeeEnvQuery(apigeeOrg.Org, apigeeOrg.Baseurl)
			apigeeEnvsResp := executeApigeeQuery(apigeeEnvURL, apigeeOrg.UserID, apigeeOrg.Password)
			log.Debug("Org: %s has environments: %s", apigeeOrg.Org, apigeeEnvsResp)
			json.Unmarshal(apigeeEnvsResp, &apigeeEnvs)
			for _, apigeeEnv := range apigeeEnvs {
				log.Debug("Asking for - Org : " + apigeeOrg.Org + ", Env : " + apigeeEnv + ", Dim : " + apigeeDimension + ", Time Range : " + apigeeTimeRange.String())
				apigeeMetricURL = buildApigeeMetricQuery(apigeeOrg.Baseurl, apigeeOrg.Org, apigeeEnv, apigeeTotalQuery, apigeeDimension, apigeeTimeRange)
				apigeeResp = executeApigeeQuery(apigeeMetricURL, apigeeOrg.UserID, apigeeOrg.Password)
				apigeeJSON = processApigeeJSON(apigeeResp)
				fatalIfErr(populateMetrics(i, apigeeJSON, apigeeOrg.Org))
			}
		}
	}
	log.Debug("Publishing")
	fatalIfErr(i.Publish())
}

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
