Feature: List Namespaces

  Scenario: ServiceAccount list namespace
    Given ServiceAccount has access to "10" namespaces
    Then the ServiceAccount can retrieve only the namespaces they have access to

  @proxy-auth
  Scenario: user list namespace
    Given User has access to "10" namespaces
    Then the User can retrieve only the namespaces they have access to
