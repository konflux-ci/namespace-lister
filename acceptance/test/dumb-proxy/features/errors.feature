@errors
Feature: Error Handling

  Scenario: Invalid token returns 401
    Given User is not authenticated
    Then the user request returns unauthorized

  Scenario: Authenticated user with no bindings gets empty list
    Then the user gets an empty namespace list
