/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"log"
	"time"

	"github.com/knative/test-infra/shared/mysql"
	"github.com/knative/test-infra/tools/monitoring/alert"
	"github.com/knative/test-infra/tools/monitoring/config"
	msql "github.com/knative/test-infra/tools/monitoring/mysql"
)

func main() {
	dbName := flag.String("database-name", "monitoring", "The monitoring database name")
	dbPort := flag.String("database-port", "3306", "The monitoring database port")

	dbUserSF := flag.String("database-user", "/secrets/cloudsql/monitoringdb/username", "Database user secret file")
	dbPassSF := flag.String("database-password", "/secrets/cloudsql/monitoringdb/password", "Database password secret file")
	dbHost := flag.String("database-host", "/secrets/cloudsql/monitoringdb/host", "Database host secret file")

	dbConfig, err := mysql.ConfigureDB(*dbUserSF, *dbPassSF, *dbHost, *dbPort, *dbName)
	if err != nil {
		log.Fatal(err)
	}

	db, err := msql.NewDB(dbConfig)
	if err != nil {
		log.Fatal(err)
	}

	c, err := config.ParseYaml(alert.YamlURL)
	if err != nil {
		log.Fatalf("Failed to parse the config yaml: %v", err)
	}

	alerts, err := db.ListAlerts()
	if err != nil {
		log.Fatalf("Failed to list the alerts: %v", err)
	}

	for _, a := range alerts {
		isAlerting := false
		isExpired := false
		alertConds := c.GetPatternAlertConditions(a.ErrorPattern)
		for job, cond := range alertConds {
			isJobAlerting, err := db.IsPatternAlerting(a.ErrorPattern, job, cond.Duration(), cond.Occurrences, cond.JobsAffected, cond.PrsAffected)
			if err != nil {
				log.Fatalf("Failed to get pattern alert status: %v", err)
			}

			isAlerting = isAlerting || isJobAlerting
			isExpired = isExpired || a.Sent.Add(cond.Duration()).Before(time.Now())
		}

		if !isAlerting || isExpired {
			db.DeleteAlert(a.ErrorPattern)
		}
	}
}
