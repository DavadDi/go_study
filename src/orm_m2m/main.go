package main

import (
	"time"

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
	num, err := m2m.Add(role, time.Now().Format("2006-01-02 15:04:05"))

	fmt.Println("num: ", num)

	if err != nil {
		fmt.Println(err)
	}

	var maps []orm.Params
	o.QueryTable("user_roles").Filter("user_id", user.Id).Values(&maps)
	if err == nil {
		fmt.Printf("Result Nums: %d\n", num)
		for _, m := range maps {
			fmt.Println(m["Id"], m["User"], m["Role"], m["Added"])
		}
	}

}
