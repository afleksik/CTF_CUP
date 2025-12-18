package middleware

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

func IsAdmin(role string) bool {
	return role == RoleAdmin
}

func IsMember(role string) bool {
	return role == RoleMember
}

func CanModifyRealm(role string) bool {
	return role == RoleAdmin
}

func CanAddUsers(role string) bool {
	return role == RoleAdmin
}

func CanCreateAssets(role string) bool {
	return role == RoleAdmin || role == RoleMember
}

func CanModifyAsset(role string, assetOwnerID string, userID string) bool {
	if role == RoleAdmin {
		return true
	}
	if role == RoleMember {
		return assetOwnerID == userID
	}
	return false
}

func CanDeleteAsset(role string, assetOwnerID string, userID string) bool {
	return CanModifyAsset(role, assetOwnerID, userID)
}
