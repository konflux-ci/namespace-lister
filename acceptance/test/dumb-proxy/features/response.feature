@response
Feature: Response Format

  Scenario: Response is a valid NamespaceList
    Given ServiceAccount has access to "3" namespaces
    Then the ServiceAccount can retrieve only the namespaces they have access to
    Then the response is a valid NamespaceList
