package content

import (
	"github.com/opesun/hypecms/api/context"
	"github.com/opesun/hypecms/model/basic"
	"github.com/opesun/hypecms/model/scut"
	"github.com/opesun/hypecms/modules/content/model"
	"github.com/opesun/hypecms/modules/user"
	"github.com/opesun/jsonp"
	//"labix.org/v2/mgo"
	"fmt"
	"labix.org/v2/mgo/bson"
	"strings"
)

var Hooks = map[string]interface{}{
	"AD":        AD,
	"Front":     Front,
	"Back":      Back,
	"Install":   Install,
	"Uninstall": Uninstall,
	"Test":      Test,
}

func Test(uni *context.Uni) error {
	front := jsonp.HasVal(uni.Opt, "Hooks.Front", "content")
	if !front {
		return fmt.Errorf("Not subscribed to front hook.")
	}
	return nil
}

func Install(uni *context.Uni, id bson.ObjectId) error {
	return content_model.Install(uni.Db, id)
}

func Uninstall(uni *context.Uni, id bson.ObjectId) error {
	return content_model.Uninstall(uni.Db, id)
}

func SaveConfig(uni *context.Uni) error {
	// id := scut.CreateOptCopy(uni.Db)
	return nil
}

func prepareOp(uni *context.Uni, op string) (bson.ObjectId, string, error) {
	typ_s, hastype := uni.Req.Form["type"]
	if !hastype {
		return "", "", fmt.Errorf("No type when doing content op %v.", op)
	}
	typ := typ_s[0]
	uid, has_uid := jsonp.Get(uni.Dat, "_user._id")
	if !has_uid {
		return "", typ, fmt.Errorf("Can't %v content, you have no id.", op)
	}
	type_opt, _ := jsonp.GetM(uni.Opt, "Modules.content.types."+typ)
	user_level := scut.Ulev(uni.Dat["_user"])
	allowed_err := content_model.AllowsContent(uni.Db, uni.Req.Form, type_opt, uid.(bson.ObjectId), user_level, op)
	if allowed_err != nil {
		return "", typ, allowed_err
	}
	return uid.(bson.ObjectId), typ, nil
}

// We never update drafts.
func SaveDraft(uni *context.Uni) error {
	post := uni.Req.Form
	typ_s, has_typ := post["type"]
	if !has_typ {
		return fmt.Errorf("No type when saving draft.")
	}
	typ := typ_s[0]
	content_type_opt, has_opt := jsonp.GetM(uni.Opt, "Modules.content.types."+typ)
	if !has_opt {
		return fmt.Errorf("Can't find options of content type %v.", typ)
	}
	allows := content_model.AllowsDraft(content_type_opt, scut.Ulev(uni.Dat["_user"]), typ)
	if allows != nil {
		return allows
	}
	rules, has_rules := jsonp.GetM(uni.Opt, "Modules.content.types."+typ+".rules")
	if !has_rules {
		return fmt.Errorf("Can't find rules of content type %v.", typ)
	}
	draft_id, err := content_model.SaveDraft(uni.Db, rules, map[string][]string(post))
	// Handle redirect.
	referer := uni.Req.Referer()
	is_admin := strings.Index(referer, "admin") != -1
	var redir string
	if err == nil { // Go to the fresh draft if we succeeded to save it.
		redir = "/content/edit/" + typ + "_draft/" + draft_id.Hex()
	} else { // Go back to the previous draft if we couldn't save the new one, or to the insert page if we tried to save a parentless draft.
		draft_id, has_draft_id := uni.Req.Form[content_model.Parent_draft_field]
		if has_draft_id && len(draft_id[0]) > 0 {
			redir = "/content/edit/" + typ + "_draft/" + draft_id[0]
		} else if id, has_id := uni.Req.Form["id"]; has_id {
			redir = "/content/edit/" + typ + "/" + id[0]
		} else {
			redir = "/content/edit/" + typ + "_draft/"
		}
	}
	if is_admin {
		redir = "/admin" + redir
	}
	uni.Dat["redirect"] = redir
	return err
}

// TODO: Move Ins, Upd, Del to other package since they can be used with all modules similar to content.
func Insert(uni *context.Uni) error {
	uid, typ, prep_err := prepareOp(uni, "insert")
	if prep_err != nil {
		return prep_err
	}
	rule, hasrule := jsonp.Get(uni.Opt, "Modules.content.types."+typ+".rules")
	if !hasrule {
		return fmt.Errorf("Can't find content type rules " + typ)
	}
	id, err := content_model.Insert(uni.Db, uni.Ev, rule.(map[string]interface{}), uni.Req.Form, uid)
	if err != nil {
		return err
	}
	// Handling redirect.
	is_admin := strings.Index(uni.Req.Referer(), "admin") != -1
	redir := "/content/edit/" + typ + "/" + id.Hex()
	if is_admin {
		redir = "/admin" + redir
	}
	uni.Dat["redirect"] = redir
	return nil
}

// TODO: Separate the shared processes of Insert/Update (type and rule checking, extracting)
func Update(uni *context.Uni) error {
	uid, typ, prep_err := prepareOp(uni, "insert")
	if prep_err != nil {
		return prep_err
	}
	rule, hasrule := jsonp.Get(uni.Opt, "Modules.content.types."+typ+".rules")
	if !hasrule {
		return fmt.Errorf("Can't find content type rules " + typ)
	}
	err := content_model.Update(uni.Db, uni.Ev, rule.(map[string]interface{}), uni.Req.Form, uid)
	if err != nil {
		return err
	}
	// We must set redirect because it can come from draft edit too.
	is_admin := strings.Index(uni.Req.Referer(), "admin") != -1
	redir := "/content/edit/" + typ + "/" + basic.StripId(uni.Req.Form["id"][0])
	if is_admin {
		redir = "/admin" + redir
	}
	uni.Dat["redirect"] = redir
	return nil
}

func Delete(uni *context.Uni) error {
	uid, _, prep_err := prepareOp(uni, "insert")
	if prep_err != nil {
		return prep_err
	}
	id, has := uni.Req.Form["id"]
	if !has {
		return fmt.Errorf("No id sent from form when deleting content.")
	}
	return content_model.Delete(uni.Db, uni.Ev, id, uid)[0] // HACK for now.
}

// Defaults to 100.
func AllowsComment(uni *context.Uni, inp map[string][]string, user_level int, op string) (string, error) {
	typ_s, has_typ := inp["type"]
	if !has_typ {
		return "", fmt.Errorf("Can't find content type when commenting.")
	}
	typ := typ_s[0]
	cont_opt, has := jsonp.GetM(uni.Opt, "Modules.content.types."+typ)
	if !has {
		return "", fmt.Errorf("Can't find options for content type %v.", typ)
	}
	var user_id bson.ObjectId
	user_id_i, has := jsonp.Get(uni.Dat, "_user._id")
	if has {
		user_id = user_id_i.(bson.ObjectId)
	}
	err := content_model.AllowsComment(uni.Db, inp, cont_opt, user_id, user_level, op)
	return typ, err
}

func InsertComment(uni *context.Uni) error {
	inp := uni.Req.Form
	user_level := scut.Ulev(uni.Dat["_user"])
	typ, allow_err := AllowsComment(uni, inp, user_level, "insert")
	if allow_err != nil {
		return allow_err
	}
	if user_level == -1 {
		err := user.PuzzleSolved(uni, "content.types.blog.comment_insert")
		if err != nil {
			return err
		}
		err = user.RegLoginBuild(uni)
		if err != nil {
			return err
		}
	}
	uid, has_uid := jsonp.Get(uni.Dat, "_user._id")
	if !has_uid {
		return fmt.Errorf("You must have user id to comment.")
	}
	user_id := uid.(bson.ObjectId)
	comment_rule, hasrule := jsonp.GetM(uni.Opt, "Modules.content.types."+typ+".comment_rules")
	if !hasrule {
		return fmt.Errorf("Can't find comment rules of content type " + typ)
	}
	mf, has := jsonp.GetI(uni.Opt, "Modules.content.types."+typ+".moderate_comment")
	moderate_first := has && mf < user_level
	if moderate_first {
		uni.Dat["_cont"] = map[string]interface{}{"awaits-moderation": true}
	}
	return content_model.InsertComment(uni.Db, uni.Ev, comment_rule, inp, user_id, moderate_first)
}

func UpdateComment(uni *context.Uni) error {
	inp := uni.Req.Form
	user_level := scut.Ulev(uni.Dat["_user"])
	typ, allow_err := AllowsComment(uni, inp, user_level, "update")
	if allow_err != nil {
		return allow_err
	}
	comment_rule, hasrule := jsonp.GetM(uni.Opt, "Modules.content.types."+typ+".comment_rules")
	if !hasrule {
		return fmt.Errorf("Can't find comment rules of content type " + typ)
	}
	uid, has_uid := jsonp.Get(uni.Dat, "_user._id")
	if !has_uid {
		return fmt.Errorf("Can't update comment, you have no id.")
	}
	return content_model.UpdateComment(uni.Db, uni.Ev, comment_rule, inp, uid.(bson.ObjectId))
}

func DeleteComment(uni *context.Uni) error {
	user_level := scut.Ulev(uni.Dat["_user"])
	_, allow_err := AllowsComment(uni, uni.Req.Form, user_level, "delete")
	if allow_err != nil {
		return allow_err
	}
	uid, has_uid := jsonp.Get(uni.Dat, "_user._id")
	if !has_uid {
		return fmt.Errorf("Can't delete comment, you have no id.")
	}
	return content_model.DeleteComment(uni.Db, uni.Ev, uni.Req.Form, uid.(bson.ObjectId))
}

func MoveToFinal(uni *context.Uni) error {
	return nil
}

func PullTags(uni *context.Uni) error {
	_, _, err := prepareOp(uni, "update")
	if err != nil {
		return err
	}
	content_id := uni.Req.Form["id"][0]
	tag_id := uni.Req.Form["tag_id"][0]
	return content_model.PullTags(uni.Db, content_id, []string{tag_id})

}

func deleteTag(uni *context.Uni) error {
	if scut.Ulev(uni.Dat["_user"]) < 300 {
		return fmt.Errorf("Only an admin can delete a tag.")
	}
	tag_id := uni.Req.Form["tag_id"][0]
	return content_model.DeleteTag(uni.Db, tag_id)
}

func SaveTypeConfig(uni *context.Uni) error {
	// id := scut.CreateOptCopy(uni.Db)
	return nil // Temp.
	return content_model.SaveTypeConfig(uni.Db, map[string][]string(uni.Req.Form))
}

// TODO: Ugly name.
func SavePersonalTypeConfig(uni *context.Uni) error {
	return nil // Temp.
	user_id_i, has := jsonp.Get(uni.Dat, "_user._id")
	if !has {
		return fmt.Errorf("Can't find user id.")
	}
	user_id := user_id_i.(bson.ObjectId)
	return content_model.SavePersonalTypeConfig(uni.Db, map[string][]string(uni.Req.Form), user_id)
}

func minLev(opt map[string]interface{}, op string) int {
	if v, ok := jsonp.Get(opt, "Modules.content."+op+"_level"); ok {
		return int(v.(float64))
	}
	return 300 // This is sparta.
}

func Back(uni *context.Uni, action string) error {
	_, ok := jsonp.Get(uni.Opt, "Modules.content")
	if !ok {
		return fmt.Errorf("No content options.")
	}
	var r error
	switch action {
	case "insert":
		if _, is_draft := uni.Req.Form["draft"]; is_draft {
			r = SaveDraft(uni)
		} else {
			r = Insert(uni)
		}
	case "update":
		if _, is_draft := uni.Req.Form["draft"]; is_draft {
			r = SaveDraft(uni)
		} else {
			r = Update(uni)
		}
	case "delete":
		r = Delete(uni)
	case "insert_comment":
		r = InsertComment(uni)
	case "update_comment":
		r = UpdateComment(uni)
	case "delete_comment":
		r = DeleteComment(uni)
	case "save_config":
		r = SaveTypeConfig(uni)
	case "pull_tags":
		r = PullTags(uni)
	case "delete_tag":
		r = deleteTag(uni)
	case "move_to_final":
		r = MoveToFinal(uni)
	case "save_type_config":
		r = SaveTypeConfig(uni)
	case "save_personal_type_config":
		r = SavePersonalTypeConfig(uni)
	default:
		return fmt.Errorf("Can't find action named \"" + action + "\" in user module.")
	}
	return r
}
