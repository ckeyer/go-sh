package admin

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ckeyer/sloth/global"
	"github.com/ckeyer/sloth/types"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type User types.User

func GetUser(db *mgo.Database, id bson.ObjectId) (*User, error) {
	u := &User{}
	err := db.C(global.ColUser).FindId(id).One(u)
	if err != nil {
		log.Errorf("find user by %s failed, %s", id, err)
		return nil, err
	}

	return u, nil
}

func (u *User) Registry(db *mgo.Database) (*User, error) {
	u.ID = bson.NewObjectId()
	u.Created = time.Now()
	u.Updated = time.Now()
	u.Role = types.RoleMember

	status, err := Status(db)
	if err != nil {
		return nil, err
	}
	if status["user"].(int) == 0 {
		u.Role = types.RoleAdmin
	}

	passwd, err := u.Password.Generate()
	if err != nil {
		log.Errorf("generate user's password %+v failed, %s", u.Password, err)
		return nil, err
	}
	u.Password = passwd

	err = db.C(global.ColUser).Insert(u)
	if err != nil {
		log.Errorf("insert user %+v failed, %s", u, err)
		return nil, err
	}

	return u, nil
}

func (u *User) Login(db *mgo.Database) (*User, error) {
	exUser := &User{}
	query := bson.M{}
	if u.Email != "" {
		query["email"] = u.Email
	} else if u.Phone != "" {
		query["phone"] = u.Phone
	} else if u.Name != "" {
		query["name"] = u.Name
	} else {
		return nil, fmt.Errorf("invalid email or phone")
	}

	err := db.C(global.ColUser).Find(query).One(exUser)
	if err != nil {
		log.Warnf("can not found user %+v in mgodb, %s", query, err)
		return nil, err
	}

	err = exUser.Password.Compare(u.Password.Bytes())
	if err != nil {
		log.Warnf("compare password failed, %s", err)
		return nil, err
	}

	exUser.LastLogin = time.Now()
	err = db.C(global.ColUser).UpdateId(exUser.ID, bson.M{
		"$set": bson.M{
			"last_login": exUser.LastLogin,
		},
	})
	if err != nil {
		log.Errorf("update login time failed, %s", err)
		return nil, err
	}

	return exUser, nil
}

func (u *User) IsAdmin() bool {
	return u.Role == types.RoleAdmin
}
