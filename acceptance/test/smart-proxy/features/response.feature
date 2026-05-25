@response
Feature: Response Format

  Scenario: Response is a valid NamespaceList
    Given User has access to "3" namespaces
    Then the User can retrieve only the namespaces they have access to
    Then the response is a valid NamespaceList
