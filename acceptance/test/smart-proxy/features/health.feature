@health
Feature: Health Endpoints

  Scenario: Liveness endpoint returns 200
    Then the healthz endpoint returns 200

  Scenario: Readiness endpoint returns 200
    Then the readyz endpoint returns 200
