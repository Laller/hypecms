// Package template_editor implements a minimalistic but idiomatic plugin for HypeCMS.
package template_editor

import (
	"github.com/opesun/hypecms/api/context"
	"github.com/opesun/hypecms/api/scut"
	//"github.com/opesun/jsonp"
	"github.com/opesun/routep"
	"labix.org/v2/mgo/bson"
	"io/ioutil"
	"fmt"
	"path/filepath"
	"strings"
	"github.com/opesun/hypecms/modules/template_editor/model"
)

// mod.GetHook accesses certain functions dynamically trough this.
var Hooks = map[string]func(*context.Uni) error {
	"Back":      Back,
	"Install":   Install,
	"Uninstall": Uninstall,
	"Test":      Test,
	"AD":        AD,
}

func NewFile(uni *context.Uni) error {
	return template_editor_model.NewFile(uni.Opt, map[string][]string(uni.Req.Form), uni.Root, uni.Req.Host)
}

func SaveFile(uni *context.Uni) error {
	return template_editor_model.SaveFile(uni.Opt, map[string][]string(uni.Req.Form), uni.Root, uni.Req.Host)
}

func DeleteFile(uni *context.Uni) error {
	return template_editor_model.DeleteFile(uni.Opt, map[string][]string(uni.Req.Form), uni.Root, uni.Req.Host)
}

func ForkPublic(uni *context.Uni) error {
	return template_editor_model.ForkPublic(uni.Db, uni.Opt, uni.Req.Host, uni.Root)
}

// main.runBackHooks invokes this trough mod.GetHook.
func Back(uni *context.Uni) error {
	if scut.NotAdmin(uni.Dat["_user"]) {
		return fmt.Errorf("You have no rights to do that.")
	}
	var r error
	action := uni.Dat["_action"].(string)
	switch action {
	case "new_file":
		r = NewFile(uni)
	case "save_file":
		r = SaveFile(uni)
	case "delete_file":
		r = DeleteFile(uni)
	case "fork_public":
		r = ForkPublic(uni)
	default:
		return fmt.Errorf("Unkown action at template_editor.")
	}
	return r
}

type Breadc struct {
	Name string
	Path string
}

func createBreadCrumb(fs []string) []Breadc {
	ret := []Breadc{}
	for i:=1; i<len(fs); i++ {
		str := strings.Replace(filepath.Join(fs[:i+1]...), "\\", "/", -1)
		ret = append(ret, Breadc{fs[i], "/" + str})
	}
	return ret
}

func View(uni *context.Uni) error {
	uni.Dat["_points"] = []string{"template_editor/view"}
	filepath_s, has := uni.Req.Form["file"]
	if !has {
		uni.Dat["error"] = "Can't find file parameter."
		return nil
	}
	filepath_str := filepath_s[0]
	tpath := scut.GetTPath(uni.Opt, uni.Req.Host)
	uni.Dat["template_name"] = scut.TemplateName(uni.Opt)
	uni.Dat["breadcrumb"] = createBreadCrumb(strings.Split(filepath_str, "/"))
	uni.Dat["can_modify"] = template_editor_model.CanModifyTemplate(uni.Opt)
	uni.Dat["filepath"] = filepath.Join(tpath, filepath_str)
	uni.Dat["raw_path"] = filepath_str
	if template_editor_model.IsDir(filepath_str) {
		fileinfos, read_err := ioutil.ReadDir(filepath.Join(uni.Root, tpath, filepath_str))
		if read_err != nil {
			uni.Dat["error"] = read_err.Error()
		}
		uni.Dat["dir"] = fileinfos
		uni.Dat["is_dir"] = true
	} else {
		file_b, read_err := ioutil.ReadFile(filepath.Join(uni.Root, tpath, filepath_str))
		if read_err != nil {
			uni.Dat["error"] = "Can't find specified file."
			return nil
		}
		if len(file_b) == 0 {
			uni.Dat["file"] = "[Empty file.]"		// A temporary hack, because the codemirror editor is not displayed when editing an empty file. It is definitely a client-side javascript problem.
		} else {
			uni.Dat["file"] = string(file_b)
		}
	}
	return nil
}

func Index(uni *context.Uni) error {
	uni.Dat["_points"] = []string{"template_editor/index"}
	return nil
}

// admin.AD invokes this trough mod.GetHook.
func AD(uni *context.Uni) error {
	ma, err := routep.Comp("/admin/template_editor/{view}", uni.P)
	if err != nil { return err }
	var r error
	switch ma["view"] {
		case "":
			r = Index(uni)
		case "view":
			r = View(uni)
		default:
			return fmt.Errorf("Unkown view at template_editor admin.")
	}
	return r
}

// admin.Install invokes this trough mod.GetHook.
func Install(uni *context.Uni) error {
	id := uni.Dat["_option_id"].(bson.ObjectId)
	return template_editor_model.Install(uni.Db, id)
}

// Admin Install invokes this trough mod.GetHook.
func Uninstall(uni *context.Uni) error {
	id := uni.Dat["_option_id"].(bson.ObjectId)
	return template_editor_model.Uninstall(uni.Db, id)
}

// main.runDebug invokes this trough mod.GetHook.
func Test(uni *context.Uni) error {
	return nil
}
