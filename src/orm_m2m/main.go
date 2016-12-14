package main

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"

	"orm_m2m/model"

	"fmt"
)

func init() {
	orm.RegisterDriver("mysql", orm.DRMySQL) //=> DR_MySQL ==> orm.DRMySQL
	dbUrl := "root:BettyJoy@1984@tcp(127.0.0.1:3306)/passport?charset=utf8"
	orm.RegisterDataBase("default", "mysql", dbUrl)
}

func main() {

	orm.Debug = true

	if err := orm.RunSyncdb("default", false, true); err != nil {
		fmt.Println(err)
		return
	}

	user := new(model.User)
	user.Name = "OrmUser"

	o := orm.NewOrm()
	o.Insert(user)

	role := new(model.Role)
	role.Id = 1
	role.Name = "RoleTest"

	o.Insert(role)

	fmt.Println("%v", user)

	_ = "breakpoint"

	m2m := o.QueryM2M(user, "Roles")
	num, err := m2m.Add(role)

	fmt.Println("num: ", num)

	if err != nil {
		fmt.Println(err)
	}

}
