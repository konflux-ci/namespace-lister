package main

type ContextKey string

const (
	ContextKeyLogger      ContextKey = "logger"
	ContextKeyUserDetails ContextKey = "user-details"
)
