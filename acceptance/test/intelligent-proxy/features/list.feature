Feature: List Namespaces

  Scenario: user list namespace
    Given User has access to "10" namespaces
    Then the User can retrieve only the namespaces they have access to
