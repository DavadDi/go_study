package main

import (
	"fmt"
	//"time"
	//"testing"
	"encoding/json"
)

type Result struct {
	Rc  uint16
	Val interface{}
}

type Token struct {
	Ysid          string
	Ytid          string
	UserId        uint
	Ts            string
	Authorization string
}

func main() {
	data := []byte(`{
		"Rc": 0,
	    "Val": {
	        "Token": {
	            "Ysid": "101179e3a6474cfda3f9eb0d4d5f20e6",
	            "Ytid": "5b5ce740f7554de0aae965e5887790c3",
	            "UserId": 5,
	            "Ts": "2016-04-12T17:56:45.598136803+08:00",
	            "Authorization": "5b5ce740f7554de0aae965e5887790c3:101179e3a6474cfda3f9eb0d4d5f20e6"
	        },
			
	        "User": {
	            "Id": 5,
	            "Name": "dave",
	            "Mobiles": [
	            {
	                "Id": 5,
	                "Mobile": "13312345629",
	                "Status": 2
	            },
				
				{
	                "Id": 6,
	                "Mobile": "13312345628",
	                "Status": 2
	            }
	            ]
	        }
	    }
	}`)

	res := Result{}

	if err := json.Unmarshal(data, &res); err != nil {
		fmt.Println("json.Unmarshal Error %s", err.Error())
		return
	}

	fmt.Printf("%v", res.Val)

	jsonObj, ok := res.Val.(map[string]interface{})
	if !ok {
		return
	}

	tokenVal := jsonObj["Token"]
	// fmt.Printf("\n\ntoken %v", tokenVal)

	token, ok := tokenVal.(map[string]interface{})

	if !ok {
		return
	}

	ysid := token["Ysid"]
	fmt.Printf("\n\n ysid %s", ysid)

	user, ok2 := jsonObj["User"].(map[string]interface{})
	if !ok2 {
		return
	}

	_ = user

	mobiles := user["Mobiles"]

	array, _ := mobiles.([]interface{})
	fmt.Printf("\n %T len %d", interface{}(mobiles), len(array)) // []interface{}

	/*
		var r interface{}
		if err := json.Unmarshal(data, &r); err != nil {
			fmt.Println("json.Unmarshal Error %s", err.Error())
		} else {
			jsonObj, ok := r.(map[string]interface{})
			if ok {
				rc := jsonObj["Rc"]
				fmt.Printf("Rc %v ", )

				val := jsonObj["Val"]
				fmt.Printf("Val %v", val)
			}
		}
	*/
}
