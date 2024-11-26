Feature: List Namespaces

  Scenario: ServiceAccount get namespace
    Given ServiceAccount has access to a namespace
    Then the ServiceAccount can retrieve the namespace
