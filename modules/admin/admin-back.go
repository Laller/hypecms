// This package implements basic admin functionality.
// - Admin login, or even register if the site has no admin.
// - Installation/uninstallation of modules.
// - Editing of the currently used options document (available under uni.Opts)
// - A view containing links to installed modules.
package admin

import (
	"encoding/json"
	"github.com/opesun/hypecms/api/context"
	"github.com/opesun/hypecms/api/mod"
	"github.com/opesun/hypecms/api/scut"
	"github.com/opesun/hypecms/modules/user"
	"github.com/opesun/jsonp"
	"github.com/opesun/routep"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
	"time"
	"runtime/debug"
	"fmt"
)

type m map[string]interface{}

func adErr(uni *context.Uni) {
	if r := recover(); r != nil {
		uni.Put("There was an error running the admin module.\n", r)
		debug.PrintStack()
	}
}

func SiteHasAdmin(db *mgo.Database) bool {
	var v interface{}
	db.C("users").Find(m{"level": m{"$gt": 299}}).One(&v)
	return v != nil
}

func regUser(db *mgo.Database, post map[string][]string) error {
	pass, pass_ok := post["password"]
	pass_again, pass_again_ok := post["password_again"]
	if !pass_ok || !pass_again_ok || len(pass) < 1 || len(pass_again) < 1 || pass[0] != pass_again[0] {
		return fmt.Errorf("improper passwords")
	} else {
		a := bson.M{"name": "admin", "level": 300, "password": pass[0]}
		err := db.C("users").Insert(a)
		if err != nil {
			return fmt.Errorf("name is not unique")
		}
	}
	return nil
}

// Registering yourself as admin is possible if the site has no admin yet.
func RegAdmin(uni *context.Uni) error {
	if SiteHasAdmin(uni.Db) {
		return fmt.Errorf("site already has an admin")
	}
	return regUser(uni.Db, uni.Req.Form)
}

func RegUser(uni *context.Uni) error {
	if !requireLev(uni.Dat["_user"], 300) {
		return fmt.Errorf("No rights")
	}
	return regUser(uni.Db, uni.Req.Form)
}

func Login(uni *context.Uni) error {
	return user.Login(uni)
}

func Logout(uni *context.Uni) error {
	return user.Logout(uni)
}

func requireLev(usr interface{}, lev int) bool {
	if val, ok := jsonp.GetI(usr, "level"); ok {
		if val >= lev {
			return true
		}
		return false
	}
	return false
}

func SaveConfig(uni *context.Uni) error {
	if !requireLev(uni.Dat["_user"], 300) {
		return fmt.Errorf("No rights to update options collection.")
	}
	jsonenc, ok := uni.Req.Form["option"]
	if ok {
		if len(jsonenc) == 1 {
			var v interface{}
			json.Unmarshal([]byte(jsonenc[0]), &v)
			if v != nil {
				m := v.(map[string]interface{})
				// Just in case
				delete(m, "_id")
				m["created"] = time.Now().Unix()
				uni.Db.C("options").Insert(m)
			} else {
				return fmt.Errorf("Invalid json.")
			}
		} else {
			return fmt.Errorf("Multiple option strings received.")
		}
	} else {
		return fmt.Errorf("No option string received.")
	}
	return nil
}

// InstallB handles both installing and uninstalling.
func InstallB(uni *context.Uni) error {
	if !requireLev(uni.Dat["_user"], 300) {
		return fmt.Errorf("No rights")
	}
	mode := ""
	if _, k := uni.Dat["_uninstall"]; k {
		mode = "uninstall"
	} else {
		mode = "install"
	}
	ma, err := routep.Comp("/admin/b/" + mode + "/{modulename}", uni.P)
	if err != nil {
		return fmt.Errorf("Bad url at " + mode)
	}
	modn, has := ma["modulename"]
	if !has {
		return fmt.Errorf("No modulename at " + mode)
	}
	if _, already := jsonp.Get(uni.Opt, "Modules." + modn); mode == "install" && already {
		return fmt.Errorf("Module " + modn + " is already installed.")
	} else if mode == "uninstall" && !already {
		return fmt.Errorf("Module " + modn + " is not installed.")
	} else {
		h := mod.GetHook(modn, strings.Title(mode))
		uni.Dat["_option_id"] = scut.CreateOptCopy(uni.Db)
		if h != nil {
			inst_err := h(uni)
			if inst_err != nil {
				return inst_err
			}
		} else {
			return fmt.Errorf("Module " + modn + " does not export the Hook " + mode + ".")
		}
	}
	return nil
}

func AB(uni *context.Uni) error {
	action := uni.Dat["_action"].(string)
	var r error
	switch action {
	case "regadmin":
		r = RegAdmin(uni)
	case "reguser":
		r = RegUser(uni)
	case "adminlogin":
		r = Login(uni)
	case "logout":
		r = Logout(uni)
	case "save-config":
		r = SaveConfig(uni)
	case "install":
		r = InstallB(uni)
	case "uninstall":
		uni.Dat["_uninstall"] = true
		r = InstallB(uni)
	default:
		return fmt.Errorf("Unknown admin action.")
	}
	return r
}
