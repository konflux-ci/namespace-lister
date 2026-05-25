@errors
Feature: Error Handling

  Scenario: Authenticated user with no bindings gets empty list
    Given User has access to "0" namespaces
    Then the user gets an empty namespace list
