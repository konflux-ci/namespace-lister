Feature: List Namespaces

  Scenario: user list namespaces
    Given User has access to "10" namespaces
    Then the User can retrieve only the namespaces they have access to

  Scenario: user not authenticated
    Given User is not authenticated
    Then  the User request is rejected with unauthorized error
