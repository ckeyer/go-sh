package admin

import (
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func NewUser(name, email, password string) {
	psd, _ := bcrypt.GenerateFromPassword(password, 13)
	return &User{
		Id:       bson.NewObjectId(),
		Name:     name,
		Email:    email,
		Password: psd,
	}

}

func (u *User) Hi() {

}