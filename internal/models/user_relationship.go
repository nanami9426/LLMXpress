package models

type UserRelationship struct {
	RelationshipID int64 `gorm:"primarykey"`
	OwnerID        int64
	TargetID       int64
	Type           int
	Basic
}

func (u *UserRelationship) TableName() string {
	return "user_relationship"
}
