package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strconv"
)

var log = logf.Log.WithName("db-connector")

func GetConnection() *sql.DB {
	log.V(2).Info("start creating connection to DB")
	host := getEnvOrFatal("DB_HOST")
	port := getEnvOrFatal("DB_PORT")
	name := getEnvOrFatal("DB_NAME")
	user := getEnvOrFatal("DB_USER")
	pass := getEnvOrFatal("DB_PASS")
	ssl := getEnvOrFatal("DB_SSL_MODE")

	conn := fmt.Sprintf("host=%v port=%v dbname=%v user=%v password=%v sslmode=%v",
		host, port, name, user, pass, ssl)

	db, err := sql.Open("postgres", conn)
	if err != nil {
		panic(fmt.Sprintf("couldn't connect to DB %v", err))
	}

	maxOpen := getIntEnvOrDefault("DB_MAX_OPEN_CONN", "5")
	maxIdle := getIntEnvOrDefault("DB_MAX_IDLE_CONN", "5")

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	log.Info("connection to DB has been established",
		"host", host, "port", port, "name", name)
	return db
}

func getEnvOrFatal(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("env variable %v is missing", key))
	}
	return value
}

func getIntEnvOrDefault(key, defaultValue string) int {
	strVal := getEnvOrDefault(key, defaultValue)
	intVal, err := strconv.Atoi(strVal)
	if err != nil {
		panic(fmt.Sprintf("cannot convert env value %v to int", strVal))
	}
	return intVal
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
