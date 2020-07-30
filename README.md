[![Experimental header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Experimental.png)](https://opensource.newrelic.com/oss-category/#experimental)

# nri-apigee

Reports Apigee performance metrics via [Apigee management API](https://docs.apigee.com/api-platform/analytics/use-analytics-api-measure-api-program-performance)

## Requirements
* Infrastructure agent installed
* Access to poll the management API

## Installation

1. Download and unpack the [latest release](https://github.com/newrelic-experimental/nri-apigee/releases/tag/v1.0).
2. cd to `bin/`
3. Run install.sh (as root, or the same user who owns the infrastructure agent process)

## Configuration
All configuration options listed below are found in `nri-apigee_metrics-settings.yml` 

* **proxyURL**: Full proxy host/IP/port to be used to communicate with Apigee API endpoint **[OPTIONAL]**
* **timeRange**: Time period to evaluate metrics retrieved (in minutes). For example, 5 would equate to collecting metrics over a 5 minute period
* **dimension**: Attribute to group retrieved metrics by (i.e - apis)
* **queries**: List of counters that will be retrieved for each combination of org and environment
* **org**: Name of Apigee org to be polled - It is possible to poll multiple orgs (each org is its own stanza)
* **baseurl**: Base API Url used to collect metrics. (i.e - http://api.apigee.com:8080/v1/organizations)
* **userID**: User that has permissions to access management API
* **password**: Password of user that has permissions to access management API

[A full list of available queries and dimensions can be found here.](https://docs.apigee.com/api-platform/analytics/analytics-reference)

## Usage
To test the integration via command line run:

```
./nri-apigee -verbose
```

## Additional Info
Apigee's docs say: "After API calls are made to proxies, it takes about 10 minutes for the data to appear in dashboards, custom reports, and management API calls."

So, in order to handle that, we have hard coded a 15 minute look back interval. This interacts with the timeRange setting in `nr-apigee_metrics-settings.yml` file to average metrics over a timeframe. When the integration polls, it asks for a single data point with all data from (now - 15 minutes) to (now - 15 minutes - timeRange). The type of data returned will depend on the aggregation function used in the query.

**Example:**
The wall clock says it's 11:30 and `timeRange` is set to 5. The integration is going to poll for sum(message_count) and avg(total_response_time).

In this case, Apigee will return the total number of Apigee messages sent between 11:10 (now - 15 minutes - timeRange) and 11:15 (now - 15 minutes). Apigee will also gather the response times of all API responses sent between 11:10 and 11:15 and create a single 5 minute average.

This integration does no mathematical processing on metrics retrieved, it simply gathers them from Apigee and forwards them to New Relic.

Please also note that the out of box execution interval of the integration is 300 seconds (aka 5 minutes). If you change timeRange, you'll also need to change the execution interval in `nri-apigee_metrics-definition.yml`, otherwise you will get silly results.

## Accessing the Data
Data will be found under the **ApigeeSample** event type within Insights. Check the Data Explorer to view metrics retrieved.

## Troubleshooting
If no data appears, check the infrastructure agent logs for any errors:

`cat <infra_log_location> | grep apigee`

## Building
Golang is required to build the integration. This was built using Go 1.14.

After cloning this repository, go to the root directory and build it:

```bash
$ make build
```

The command above builds an executable file called `nri-apigee` under the `bin` directory.

To test the integration, run `nri-apigee`:

```bash
$ ./bin/nri-apigee
```

## Contributing
We encourage your contributions to improve the New Relic Infrastructure Integration for Apigee! Keep in mind when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.
If you have any questions, or to execute our corporate CLA, required if your contribution is on behalf of a company,  please drop us an email at opensource@newrelic.com.

## License
The New Relic Infrastructure Integration for Apigee is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.

The New Relic Infrastructure Integration for Apigee also uses source code from third-party libraries. You can find full details on which libraries are used and the terms under which they are licensed in the [third-party notices document](https://github.com/newrelic/nri-apigee/blob/main/THIRD_PARTY_NOTICES.md).
