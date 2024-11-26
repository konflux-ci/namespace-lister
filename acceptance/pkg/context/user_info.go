package context

import (
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type UserInfo struct {
	Token     string
	Namespace string
	Name      string
	APIGroup  string
	Kind      string
}

func UserInfoFromServiceAccount(sa corev1.ServiceAccount, tkn *authenticationv1.TokenRequest) UserInfo {
	return UserInfo{
		Namespace: sa.Namespace,
		Name:      sa.Name,
		Kind:      "ServiceAccount",
		APIGroup:  "",
		Token:     tkn.Status.Token,
	}
}

func UserInfoFromUsername(username string) UserInfo {
	return UserInfo{
		Namespace: "",
		Name:      username,
		Kind:      "User",
		APIGroup:  rbacv1.GroupName,
		Token:     "",
	}
}

func (u *UserInfo) AsSubject() rbacv1.Subject {
	return rbacv1.Subject{
		Namespace: u.Namespace,
		Name:      u.Name,
		APIGroup:  u.APIGroup,
		Kind:      u.Kind,
	}
}
