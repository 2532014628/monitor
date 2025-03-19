package model

type User struct {
	ID         int    	`json:"id" gorm:"primarykey;autoIncrement"`
	Name       string 	`json:"name" gorm:"column:name; not null"`
	Email      string 	`json:"email" gorm:"unique;not null"`
	Password   string 	`json:"password" gorm:"not null"`
	RoleId	   int 		`json:"role_id" gorm:"column:role_id;default:0"`	// 3: root, 2: company_admin, 0: user
	CompanyId  int 		`json:"company_id" gorm:"column:company_id;default:0"`
	IsVerified bool		`json:"is_verified" gorm:"column:is_verified"`
}

type Company struct{
	ID int `json:"id" gorm:"primarykey;autoIncrement"`
	Name string `json:"name" gorm:"column:name; not null"`
	Description string `json:"description" gorm:"column:description; not null"`
	AdminID int `json:"admin_id" gorm:"column:admin_id; not null"`
	MemberNum int `json:"member_num" gorm:"column:member_num; default:0"`
	ServerNum int `json:"server_num" gorm:"column:server_num; default:0"`
}