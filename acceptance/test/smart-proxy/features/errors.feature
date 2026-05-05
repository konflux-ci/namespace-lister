@errors
Feature: Error Handling

  Scenario: Authenticated user with no bindings gets empty list
    Given User has access to "0" namespaces
    Then the user gets an empty namespace list

  Scenario: Response contains valid NamespaceList fields
    Given User has access to "3" namespaces
    Then the User can retrieve only the namespaces they have access to
    Then the response is a valid NamespaceList
