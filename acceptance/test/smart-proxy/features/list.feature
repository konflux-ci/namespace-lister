Feature: List Namespaces

  Scenario: user list namespaces
    Given User has access to "10" namespaces
    Then the User can retrieve only the namespaces they have access to

  Scenario: user with groups list namespaces
    Given User has access to "10" namespaces
    Given Group "mygroup" has access to "10" namespaces
    Given User is part of group "mygroup"
    Then the User can retrieve the namespaces they and their groups have access to

  Scenario: user not authenticated
    Given User is not authenticated
    Then  the User request is rejected with unauthorized error

  Scenario: ClusterRoleBindings are ignored
    Given the ServiceAccount has Cluster-scoped get permission on namespaces
    Given 10 tenant namespaces exist
    Then the ServiceAccount retrieves no namespaces
