package models

type Group struct {
	GroupID int64 `gorm:"primarykey"`
	Name    string
	OwnerID int64
	Icon    string
	Desc    string
}

func (g *Group) TableName() string {
	return "group"
}
