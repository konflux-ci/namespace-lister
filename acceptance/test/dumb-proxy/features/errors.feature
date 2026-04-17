@errors
Feature: Error Handling

  Scenario: Invalid token returns 401
    Given User is not authenticated
    Then the user request returns unauthorized

  Scenario: Authenticated user with no bindings gets empty list
    Then the user gets an empty namespace list

  Scenario: Response format is a valid NamespaceList
    Given ServiceAccount has access to "3" namespaces
    Then the ServiceAccount can retrieve only the namespaces they have access to
    Then the response is a valid NamespaceList
