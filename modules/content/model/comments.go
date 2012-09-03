package content_model

import(
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	ifaces "github.com/opesun/hypecms/interfaces"
	"github.com/opesun/extract"
	"github.com/opesun/hypecms/model/basic"
	"fmt"
	"time"
)

func commentRequiredLevel(content_options map[string]interface{}, op string) int {
	var req_lev int
	if lev, has_lev := content_options[op + "_comment_level"]; has_lev {
		req_lev = int(lev.(float64))
	} else {
		req_lev = 100
	}
	return req_lev
}

func AllowsComment(db *mgo.Database, inp map[string][]string, content_options map[string]interface{}, user_id bson.ObjectId, user_level int, op string) error {
	_, turned_off := content_options["comments_off"]
	if turned_off {
		return fmt.Errorf("Comments are turned off currently.")
	}
	req_lev := commentRequiredLevel(content_options, op)
	if user_level < req_lev {
		return fmt.Errorf("You have no rights to comment.")
	}
	rule := map[string]interface{}{
		"content_id": 	"must",
		"comment_id":	"must",
		"type":			"must",
	}
	dat, err := extract.New(rule).Extract(inp)
	if err != nil { return err }
	content_id_str := basic.StripId(dat["content_id"].(string))
	typ := dat["type"].(string)		// We check this because they can lie about the type, sending a less strictly guarded type name and gaining access.
	if !typed(db, bson.ObjectIdHex(content_id_str), typ) {
		return fmt.Errorf("Content is not of type %v.", typ)
	}
	// Even if he has the required level, and he is below level 200 (not a moderator), he can't modify other people's comment, only his owns.
	// So we query here the comment and check who is the owner of it.
	if user_level < 200 && op != "insert" {
		if user_level == 0 { return fmt.Errorf("Not registered users can't update or delete comments currently.") }
		comment_id_str := basic.StripId(dat["comment_id"].(string))
		auth, err := findCommentAuthor(db, content_id_str, comment_id_str)
		if err != nil {
			return err
		}
		if auth.Hex() != user_id.Hex() {
			return fmt.Errorf("You are not the rightous owner of the comment.")
		}
	}
	return nil
}

// To be able to list all comments chronologically we insert it to a virtual collection named "comments", where there will be only a link.
// "_id" equals to "comment_id" in the content comment array.
func insertToVirtual(db *mgo.Database, content_id, comment_id, author bson.ObjectId, in_moderation bool) error {
	comment_link := map[string]interface{}{
		"_contents_parent": content_id,
		"_id":		comment_id,
		"_users_author":	author,
		"created":			time.Now().Unix(),
	}
	return db.C("comments").Insert(comment_link)
}

// Places a comment into its final place - the comment array field of a given content.
func insertFinal(db *mgo.Database, comment map[string]interface{}, comment_id, content_id bson.ObjectId) error {
	comment["comment_id"] = comment_id
	q := bson.M{ "_id": content_id}
	upd := bson.M{
		"$inc": bson.M{
			"comment_count": 1,
		},
		"$push": bson.M{
			"comments": comment,
		},
	}
	return db.C("contents").Update(q, upd)
}

// MoveToFinal with extract.
func MoveToFinalWE(db *mgo.Database, inp map[string][]string) error {
	r := map[string]interface{}{
		"comment_id": "must",
	}
	dat, err := extract.New(r).Extract(inp)
	if err != nil { return err }
	comment_id := basic.ToIdWithCare(dat["comment_id"])
	return MoveToFinal(db, comment_id)
}

// Moves comment from moderation queue into its final place - the comment array field of a given content.
func MoveToFinal(db *mgo.Database, comment_id bson.ObjectId) error {
	var comm interface{}
	err := db.C("comments_moderation").Find(m{"_id": comment_id}).One(&comm)
	if err != nil { return err }
	comment := basic.Convert(comm).(map[string]interface{})
	comment["comment_id"] = comment["_id"]
	delete(comment, "comment_id")
	content_id := comment["_contents_parent"].(bson.ObjectId)
	q := m{"_id": content_id}
	upd := m{
		"$inc": m{
			"comment_count": 1,
		},
		"$push": m{
			"comments": comment,
		},
	}
	return db.C("contents").Update(q, upd)
}

// Puts comment coming from UI into moderation queue.
func insertModeration(db *mgo.Database, comment map[string]interface{}, comment_id, content_id bson.ObjectId) error {
	comment["_id"] = comment_id
	comment["_contents_parent"] = content_id
	return db.C("comments_moderation").Insert(comment)
}

// Apart from rule, there is one mandatory field which must come from the UI: "content_id"
// moderate_first should be read as "moderate first if it is a valid, spam protection passed comment"
// Spam protection happens outside of this anyway.
func InsertComment(db *mgo.Database, ev ifaces.Event, rule map[string]interface{}, inp map[string][]string, user_id bson.ObjectId, moderate_first bool) error {
	dat, err := extract.New(rule).Extract(inp)
	if err != nil {
		return err
	}
	basic.DateAndAuthor(rule, dat, user_id, false)
	ids, err := basic.ExtractIds(inp, []string{"content_id"})
	if err != nil {
		return err
	}
	content_id := bson.ObjectIdHex(ids[0])
	comment_id := bson.NewObjectId()
	if moderate_first {
		err = insertModeration(db, dat, comment_id, content_id)
	} else {
		err = insertFinal(db, dat, comment_id, content_id)
		if err != nil {
			err = insertToVirtual(db, content_id, comment_id, user_id, moderate_first)
		}
	}
	return err
}

// Apart from rule, there are two mandatory field which must come from the UI: "content_id" and "comment_id"
func UpdateComment(db *mgo.Database, ev ifaces.Event, rule map[string]interface{}, inp map[string][]string, user_id bson.ObjectId) error {
	dat, err := extract.New(rule).Extract(inp)
	if err != nil {
		return err
	}
	basic.DateAndAuthor(rule, dat, user_id, true)
	ids, err := basic.ExtractIds(inp, []string{"content_id", "comment_id"})
	if err != nil {
		return err
	}
	comment_id := bson.ObjectIdHex(ids[1])
	q := bson.M{
		"_id": bson.ObjectIdHex(ids[0]),
		"comments.comment_id": comment_id,
	}
	upd := bson.M{
		"$set": bson.M{
			"comments.$": dat,
		},
	}
	err = db.C("contents").Update(q, upd)
	if err != nil { return err }
	return db.C("comments").Remove(m{"_id": comment_id})
}

// Two mandatory fields must come from UI: "content_id" and "comment_id"
func DeleteComment(db *mgo.Database, ev ifaces.Event, inp map[string][]string, user_id bson.ObjectId) error {
	ids, err := basic.ExtractIds(inp, []string{"content_id", "comment_id"})
	if err != nil {
		return err
	}
	q := bson.M{
		"_id": bson.ObjectIdHex(ids[0]),
		"comments.comment_id": bson.ObjectIdHex(ids[1]),
	}
	upd := bson.M{
		"$inc":	bson.M{
			"comment_count": -1,
		},
		"$pull": bson.M{
			"comments": bson.M{
				"comment_id": bson.ObjectIdHex(ids[1]),
			},
		},
	}
	return db.C("contents").Update(q, upd)
}

func findComment(db *mgo.Database, content_id, comment_id string) (map[string]interface{}, error) {
	var v interface{}
	q := bson.M{
		"_id": bson.ObjectIdHex(content_id),
		//"comments.comment_id": bson.ObjectIdHex(comment_id),	
	}
	find_err := db.C("contents").Find(q).One(&v)
	if find_err != nil { return nil, find_err }
	if v == nil {
		return nil, fmt.Errorf("Can't find content with id %v.", content_id)
	}
	v = basic.Convert(v)
	comments_i, has := v.(map[string]interface{})["comments"]
	if !has {
		return nil, fmt.Errorf("No comments in given content.")
	}
	comments, ok := comments_i.([]interface{})
	if !ok {
		return nil, fmt.Errorf("comments member is not a slice in content %v", content_id)
	}
	// TODO: there must be a better way.
	for _, v_i := range comments {
		v, is_map := v_i.(map[string]interface{})
		if !is_map { continue }
		if val_i, has := v["comment_id"]; has {
			if val_id, ok := val_i.(bson.ObjectId); ok {
				if val_id.Hex() == comment_id {
					return v, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("Comment not found.")
}

func findCommentAuthor(db *mgo.Database, content_id, comment_id string) (bson.ObjectId, error) {
	comment, err := findComment(db, content_id, comment_id)
	if err != nil { return "", err }
	author, has := comment["created_by"]
	if !has {
		return "", fmt.Errorf("Given content has no author.")
	}
	return author.(bson.ObjectId), nil
}