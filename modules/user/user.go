// Package user implements basic user functionality.
// - Registration, deletion, update, login, logout of users.
// - Building the user itself (if logged in), and putting it to uni.Dat["_user"].
package user

import (
	"github.com/opesun/hypecms/api/context"
	"github.com/opesun/hypecms/modules/user/model"
	"github.com/opesun/jsonp"
	"net/http"
	"fmt"
)

var Hooks = map[string]func(*context.Uni) error {
	"BuildUser": BuildUser,
	"Back":      Back,
	"Test":      Test,
}

// Recover from wrong ObjectId like panics. Unset the cookie.
func unsetCookie(w http.ResponseWriter, dat map[string]interface{}, err *error) {
	r := recover(); if r == nil { return }
	*err = nil	// Just to be sure.
	c := &http.Cookie{Name: "user", Value: "", MaxAge: 3600000, Path: "/"}
	http.SetCookie(w, c)
	dat["_user"] = user_model.EmptyUser()
}

// If there were some random database query errors or something we go on with an empty user.
func BuildUser(uni *context.Uni) (err error) {
	defer unsetCookie(uni.W, uni.Dat, &err)
	var user_id_str string
	c, err := uni.Req.Cookie("user")
	if err != nil { panic(err) }
	user_id_str = c.Value
	block_key := []byte(uni.Secret())
	user_id, err := user_model.DecryptId(user_id_str, block_key)
	if err != nil { panic(err) }
	user, err := user_model.BuildUser(uni.Db, uni.Ev, user_id, uni.Req.Header)
	if err != nil { panic(err) }
	uni.Dat["_user"] = user
	fmt.Println(user)
	return
}

// Helper function to hotregister a guest user, log him in and build his user data into uni.Dat["_user"].
func RegLoginBuild(uni *context.Uni) error {
	db := uni.Db
	ev := uni.Ev
	guest_rules, _ := jsonp.GetM(uni.Opt, "Modules.user.guest_rules")	// RegksterGuest will do fine with nil.
	inp := uni.Req.Form
	http_header := uni.Req.Header
	dat := uni.Dat
	w := uni.W
	block_key := []byte(uni.Secret())
	guest_id, err := user_model.RegisterGuest(db, ev, guest_rules, inp)
	if err != nil { return err }
	_, _, err = user_model.FindLogin(db, inp)
	if err != nil { return err }
	err = user_model.Login(w, guest_id, block_key)
	if err != nil { return err }
	user, err := user_model.BuildUser(db, ev, guest_id, http_header)
	if err != nil {	return err }
	dat["_user"] = user
	return nil
}

func PuzzleSolved(uni *context.Uni) bool {
	inp := uni.Req.Form
	block_key := []byte(uni.Secret())
	return user_model.PuzzleSolved(inp, block_key)
}

func PuzzleSolvedE(uni *context.Uni) error {
	if !PuzzleSolved(uni) {
		return fmt.Errorf("Puzzle remained unsolved.")
	}
	return nil
}

func Register(uni *context.Uni) error {
	inp := uni.Req.Form
	rules, _ := jsonp.GetM(uni.Opt, "Modules.user.rules")		// RegisterUser will be fine with nil.
	_, err := user_model.RegisterUser(uni.Db, uni.Ev, rules, inp)
	return err
}

func Login(uni *context.Uni) error {
	// Maybe there could be a check here to not log in somebody who is already logged in.
	inp := uni.Req.Form
	if _, id, err := user_model.FindLogin(uni.Db, inp); err == nil {
		block_key := []byte(uni.Secret())
		return user_model.Login(uni.W, id, block_key)
	} else {
		return err
	}
	return nil
}

func Logout(uni *context.Uni) error {
	c := &http.Cookie{Name: "user", Value: "", Path: "/"}
	http.SetCookie(uni.W, c)
	return nil
}

func TestRaw(opt map[string]interface{}) map[string]interface{} {
	msg := make(map[string]interface{})
	// _, has := jsonp.Get(opt, "BuildUser")
	// msg["BuildUser"] = has
	has := jsonp.HasVal(opt, "Hooks.Back", "user")
	msg["Back"] = has
	return msg
}

func Test(uni *context.Uni) error {
	uni.Dat["_cont"] = TestRaw(uni.Opt)
	return nil
}

func Back(uni *context.Uni) error {
	action := uni.Dat["_action"].(string)
	var err error
	switch action {
	case "login":
		err = Login(uni)
	case "logout":
	case "register":
	default:
		err = fmt.Errorf("Unkown action at user module.")
	}
	return err
}
