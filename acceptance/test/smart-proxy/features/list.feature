Feature: List Namespaces

  Scenario: user list namespaces
    Given User has access to "10" namespaces
    Then the User can retrieve only the namespaces they have access to

  Scenario: user not authenticated
    Given User is not authenticated
    Then  the User request is rejected with unauthorized error

  Scenario: ClusterRoleBindings are ignored
    Given the ServiceAccount has Cluster-scoped get permission on namespaces
    Given 10 tenant namespaces exist
    Then the ServiceAccount retrieves no namespaces
