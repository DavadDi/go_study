// model.go
package model

import (
	"time"

	"github.com/astaxie/beego/orm"
)

type User struct {
	Id    uint64
	Name  string  `orm:"size(64)"`
	Roles []*Role `orm:"rel(m2m);rel_through(orm_m2m/model.UserRoles)"`
}

type Role struct {
	Id   uint64
	Name string `orm:"size(64)"`

	Users []*User `orm:"reverse(many);rel_through(orm_m2m/model.UserRoles)"`
}

type UserRoles struct {
	Id    int       `orm:"column(id)"`
	User  *User     `orm:"rel(fk);column(user_id)"`
	Role  *Role     `orm:"rel(fk);column(role_id)"`
	Added time.Time `orm:"auto_now_add;type(datetime)"`
}

func init() {
	orm.RegisterModel(new(UserRoles))
	orm.RegisterModel(new(User))
	orm.RegisterModel(new(Role))
}
