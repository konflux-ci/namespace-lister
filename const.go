package main

const (
	EnvLogLevel          string = "LOG_LEVEL"
	EnvUsernameHeader    string = "AUTH_USERNAME_HEADER"
	EnvAddress           string = "ADDRESS"
	EnvCacheResyncPeriod string = "CACHE_RESYNC_PERIOD"

	DefaultAddr           string = ":8080"
	DefaultHeaderUsername string = "X-Email"

	HttpContentType            string = "Content-Type"
	HttpContentTypeApplication string = "application/json;charset=utf-8"
)
