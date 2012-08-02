// Package scut contains a somewhat ugly but useful collection of frequently appearing patterns to allow faster prototyping.
package scut

import(
	"fmt"
	"sort"
	"labix.org/v2/mgo/bson"
	"path/filepath"
	"strings"
)

// Converts all bson.ObjectId to string. Usually called before display.
func Strify(v interface{}) {
	switch value := v.(type) {
	case bson.M:
		for i, mem := range value {
			if id, is_id := mem.(bson.ObjectId); is_id {
				value[i] = id.Hex()
			} else {
				Strify(mem)
			}
		}
	case map[string]interface{}:
		for i, mem := range value {
			if id, is_id := mem.(bson.ObjectId); is_id {
				value[i] = id.Hex()
			} else {
				Strify(mem)
			}
		}
	case []interface{}:
		for i, mem := range value {
			if id, is_id := mem.(bson.ObjectId); is_id {
				value[i] = id.Hex()
			} else {
				Strify(mem)
			}
		}
	}
}

// A more generic version of abcKeys. Takes a map[string]interface{} and puts every element of that into an []interface{}, ordered by keys alphabetically.
// TODO: find the intersecting parts between the two functions and refactor.
func OrderKeys(d map[string]interface{}) []interface{} {
	keys := []string{}
	for i, _ := range d {
		keys = append(keys, i)
	}
	sort.Strings(keys)
	ret := []interface{}{}
	for _, v := range keys {
		if ma, is_ma := d[v].(map[string]interface{}); is_ma {
			// RETHINK: What if a key field gets overwritten? Should we name it _key?
			ma["key"] = v
		}
		ret = append(ret, d[v])
	}
	return ret
}
// TODO: ret should contain the rules, so we can display/js validate based on them too.
// Extract module should be modified to not blow up when encountering an unkown rule field, so we can embed metainformation (like text or input, WYSIWYG editor, etc) in the rule too.
//
// Takes a dat map[string]interface{}, and puts every element of that which is defined in r to a slice, sorted by the keys ABC order.
// prior parameter can override the default abc ordering, so keys in prior will be the first ones in the slice, if those keys exist.
func abcKeys(rule map[string]interface{}, dat map[string]interface{}, prior []string) []map[string]interface{} {
	ret := []map[string]interface{}{}
	already_in := map[string]struct{}{}
	for _, v := range prior {
		if _, contains := rule[v]; contains {
			item := map[string]interface{}{v:1, "key":v}
			if dat != nil {
				item["value"] = dat[v]
			}
			ret = append(ret, item)
			already_in[v] = struct{}{}
		}
	}
	keys := []string{}
	for i, v := range rule {
		// If the value is not false
		if boo, is_boo := v.(bool); !is_boo || boo == true {
			keys = append(keys, i)
		}
	}
	sort.Strings(keys)
	for _, v := range keys {
		if _, in := already_in[v]; !in {
			item := map[string]interface{}{v:1, "key":v}
			if dat != nil {
				item["value"] = dat[v]
			}
			ret = append(ret, item)
		}
	}
	return ret
}

// Takes an extraction/validation rule, a document and from that creates a slice which can be easily displayed by a templating engine as a html form.
func RulesToFields(rule interface{}, dat interface{}) ([]map[string]interface{}, error) {
	rm, rm_ok := rule.(map[string]interface{})
	if !rm_ok {
		return nil, fmt.Errorf("Rule is not a map[string]interface{}.")
	}
	datm, datm_ok := dat.(map[string]interface{})
	if !datm_ok && dat != nil {
		return nil, fmt.Errorf("Dat is not a map[string]interface{}.")
	}
	return abcKeys(rm, datm, []string{"title", "name", "slug"}), nil
}

func TemplateType(opt map[string]interface{}) string {
	_, priv := opt["TplIsPrivate"]
	var ttype string
	if priv {
		ttype = "private"
	} else {
		ttype = "public"
	}
	return ttype
}

func TemplateName(opt map[string]interface{}) string {
	tpl, has_tpl := opt["Template"]
	if !has_tpl {
		tpl = "default"
	}
	return tpl.(string)
}

// Observes opt and gives you back a string describing the path of your template eg "templates/public/template_name"
func GetTPath(opt map[string]interface{}, host string) string {
	templ := TemplateName(opt)
	ttype := TemplateType(opt)
	if ttype == "public" {
		return filepath.Join("templates", ttype, templ)
	}
	return filepath.Join("templates", ttype, host, templ)
}

// Inp:	"admin/this/that.txt"
// []string{ "modules/admin/tpl", "this/that.txt"}
func GetModTPath(filename string) []string {
	sl := []string{}
	p := strings.Split(filename, "/")
	sl = append(sl, filepath.Join("modules", p[0], "tpl"))
	sl = append(sl, strings.Join(p[1:], "/"))
	return sl
}

func NotAdmin(user interface{}) bool {
	return ULev(user) < 300
}

func ULev(useri interface{}) int {
	if useri == nil {
		return 0
	}
	user := useri.(map[string]interface{})
	ulev, has := user["level"]
	if !has {
		return 0
	}
	return int(ulev.(int))
}

func Merge(a map[string]interface{}, b map[string]interface{}) {
	for i, v := range b {
		a[i] = v
	}
}