@cache @serial
Feature: Cache Consistency

  Scenario: Namespace created after startup becomes visible
    Given User has access to "3" namespaces
    Then the User can retrieve only the namespaces they have access to
    When a new namespace "dynamic-ns" is created with access for the current user
    Then the user can see namespace "dynamic-ns" in the list

  Scenario: Namespace deleted while service is running becomes invisible
    Given User has access to "3" namespaces
    When a new namespace "to-delete" is created with access for the current user
    Then the user can see namespace "to-delete" in the list
    When the namespace "to-delete" is deleted
    Then the user cannot see namespace "to-delete" in the list

  Scenario: RoleBinding added grants access to existing namespace
    Given a namespace "to-grant" exists without access for the current user
    Then the user cannot see namespace "to-grant" in the list
    When a RoleBinding granting access is added in namespace "to-grant"
    Then the user can see namespace "to-grant" in the list

  Scenario: RoleBinding removed revokes access
    Given User has access to "3" namespaces
    When a new namespace "to-revoke" is created with access for the current user
    Then the user can see namespace "to-revoke" in the list
    When the RoleBinding is removed from namespace "to-revoke"
    Then the user cannot see namespace "to-revoke" in the list

  Scenario: Bulk concurrent namespace and RBAC changes
    Given User has access to "5" namespaces
    Then the User can retrieve only the namespaces they have access to
    When 5 namespaces with access are created and 5 existing namespaces are deleted
    Then the user sees exactly 5 namespaces
