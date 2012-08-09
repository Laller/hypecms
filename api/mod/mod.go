// This package gets around the lack of dynamic code loading in Go.
// There may be a better solution then putting every exported (in terms of cms) function into a map.
package mod

import (
	"github.com/opesun/hypecms/api/context"
	"github.com/opesun/hypecms/modules/content"
	"github.com/opesun/hypecms/modules/skeleton"
	"github.com/opesun/hypecms/modules/user"
	"github.com/opesun/hypecms/modules/display_editor"
	"github.com/opesun/hypecms/modules/template_editor"
	"github.com/opesun/hypecms/modules/custom_actions"
)

func GetHook(modname string, method string) func(*context.Uni) error {
	var r func(*context.Uni) error
	switch modname {
	case "content":
		r = content.Hooks[method]
	case "user":
		r = user.Hooks[method]
	case "skeleton":
		r = skeleton.Hooks[method]
	case "custom_actions":
		r = custom_actions.Hooks[method]
	case "display_editor":
		r = display_editor.Hooks[method]
	case "template_editor":
		r = template_editor.Hooks[method]
	default:								// Such a crucial bug.
		panic("mod.Gethook cant find module " + modname)
	}
	return r
}
