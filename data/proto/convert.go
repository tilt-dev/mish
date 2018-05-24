package proto

import (
	"github.com/windmilleng/mish/data"
)

func UserP2D(user *User) data.User {
	return data.User{
		UserID:      data.UserID(user.UserId),
		Username:    data.Username(user.Username),
		Email:       user.Email,
		AvatarURL:   user.AvatarUrl,
		Name:        user.Name,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Description: user.Description,
		Location:    user.Location,
	}
}

func UserD2P(user data.User) *User {
	return &User{
		UserId:      uint64(user.UserID),
		Username:    string(user.Username),
		Email:       user.Email,
		AvatarUrl:   user.AvatarURL,
		Name:        user.Name,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Description: user.Description,
		Location:    user.Location,
	}
}
