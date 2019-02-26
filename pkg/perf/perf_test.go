package perf

import (
	"log"
	"os"
	"strconv"
)

func ExamplePerfClient_AddJobsToJenkinsDS() {
	url := lookupEnv("PERF_URL")
	user := lookupEnv("PERF_USER")
	pass := lookupEnv("PERF_PASS")
	dsId, _ := strconv.Atoi(lookupEnv("PERF_JENKINS_DS_ID"))
	cl, _ := NewRestClient(url, user, pass)

	_ = cl.AddJobsToJenkinsDS(dsId, []string{"job-1", "job-2"})
}

func ExamplePerfClient_AddProjectsToSonarDS() {
	url := lookupEnv("PERF_URL")
	user := lookupEnv("PERF_USER")
	pass := lookupEnv("PERF_PASS")
	dsId, _ := strconv.Atoi(lookupEnv("PERF_SONAR_DS_ID"))
	cl, _ := NewRestClient(url, user, pass)

	_ = cl.AddProjectsToSonarDS(dsId, []string{"pr-1", "pr-2"})
}

func ExamplePerfClient_AddProjectsToGerritDS() {
	url := lookupEnv("PERF_URL")
	user := lookupEnv("PERF_USER")
	pass := lookupEnv("PERF_PASS")
	dsId, _ := strconv.Atoi(lookupEnv("PERF_GERRIT_DS_ID"))
	cl, _ := NewRestClient(url, user, pass)

	_ = cl.AddProjectsToGerritDS(dsId, []GerritPerfConfig{{Branches: []string{"master"}}})
}

func ExamplePerfClient_AddProjectsToGitLabDS() {
	url := lookupEnv("PERF_URL")
	user := lookupEnv("PERF_USER")
	pass := lookupEnv("PERF_PASS")
	dsId, _ := strconv.Atoi(lookupEnv("PERF_GITLAB_DS_ID"))
	cl, _ := NewRestClient(url, user, pass)

	_ = cl.AddRepositoriesToGitlabDS(dsId, map[string]string{"test": "test-br"})
}

func lookupEnv(key string) string {
	value, isPresented := os.LookupEnv(key)
	if !isPresented {
		log.Fatalf("required env variable by key %s is not presented", key)
	}
	return value
}
