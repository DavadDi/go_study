Go中的 **encoding/json** 包非常方便地编码 **struct** 为 **json** 格式的数据 ;


	package main
	
	import (
		"encoding/json"
		"os"
		"time"
	)
	
	type MyUser struct {
		ID       int64     `json:"id"`
		Name     string    `json:"name"`
		LastSeen time.Time `json:"lastSeen"`
	}
	
	func main() {
		_ = json.NewEncoder(os.Stdout).Encode(
			&MyUser{1, "Ken", time.Now()},
		)
	}
	
输出为:

	{"id":1,"name":"Ken","lastSeen":"2009-11-10T23:00:00Z"}
	
但是如果我们想改变 **struct** 中的某个字段的显示怎么办？ 比如如果我们想把 **LastSeen** 显示成unix格式的timestamp。

很简单的方式就是引入一个附加的 **struct**， 然后定义接口方法 **MarshalJSON** 格式化成期望的显示格式。

	func (u *MyUser) MarshalJSON() ([]byte, error) {
		return json.Marshal(&struct {
			ID       int64  `json:"id"`
			Name     string `json:"name"`
			LastSeen int64  `json:"lastSeen"`
		}{
			ID:       u.ID,
			Name:     u.Name,
			LastSeen: u.LastSeen.Unix(),
		})
	}

以上代码可以很好的工作，但是如果我们的 **struct** 中包含比较多的字段的话，上述方式就变得比较笨重。如果我们可以将 **原有的struct** 嵌入到 **引入的struct**，让其从  **原有的struct** 中继承不需要修改格式的字段，这种方式会更加方便。

	func (u *MyUser) MarshalJSON() ([]byte, error) {
		return json.Marshal(&struct {
			LastSeen int64 `json:"lastSeen"`
			*MyUser
		}{
			LastSeen: u.LastSeen.Unix(),
			MyUser:   u,
		})
	}
	
但是问题在于，附加的 **struct** 继承 **字段** 的同时也继承了 **MarshalJSON** 方法，这样将导致死循环。解决的方式就是采用 **alias** 原有的类型。**alias** 的方式将继承原有 **struct** 中的相同的 **字段** , 但是不继承其方法。

	func (u *MyUser) MarshalJSON() ([]byte, error) {
		type Alias MyUser
		return json.Marshal(&struct {
			LastSeen int64 `json:"lastSeen"`
			*Alias
		}{
			LastSeen: u.LastSeen.Unix(),
			Alias:    (*Alias)(u),
		})
	}

同样的技巧也可以工作在方法 **UnmarshalJSON** 中。

	func (u *MyUser) UnmarshalJSON(data []byte) error {
		type Alias MyUser
		aux := &struct {
			LastSeen int64 `json:"lastSeen"`
			*Alias
		}{
			Alias: (*Alias)(u),
		}
		if err := json.Unmarshal(data, &aux); err != nil {
			return err
		}
		u.LastSeen = time.Unix(aux.LastSeen, 0)
		return nil
	}
	
	
