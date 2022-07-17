package models

import (
	"strconv"
	"strings"
	"treehole_next/config"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

type User struct {
	BaseModel
	Favorites   []Hole                 `json:"favorites" gorm:"many2many:user_favorites"`
	Roles       []string               `json:"-" gorm:"-:all"`
	BanDivision map[int]bool           `json:"-" gorm:"-:all"`
	Nickname    string                 `json:"nickname" gorm:"-:all"`
	Config      map[string]interface{} `json:"config" gorm:"-:all"`
	IsAdmin     bool                   `json:"is_admin" gorm:"-:all"`
	IsOperation bool                   `json:"is_operation" gorm:"-:all"`
}

func (user *User) GetUser(c *fiber.Ctx) error {
	id, err := GetUserID(c)
	if err != nil {
		return err
	}
	user.ID = id
	if config.Config.Debug {
		user.IsAdmin = true
		user.IsOperation = true
		return nil
	}

	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	user.Roles = claims["roles"].([]string)
	user.Nickname = claims["nickname"].(string)
	for _, v := range user.Roles {
		if v == "admin" {
			user.IsAdmin = true
		} else if v == "operation" {
			user.IsOperation = true
		} else if strings.HasPrefix(v, "ban_treehole") {
			banInfo := strings.Split(v, "_")
			banDivisionID, err := strconv.Atoi(banInfo[2])
			if err != nil {
				return err
			}
			user.BanDivision[banDivisionID] = true
		}
	}
	return nil
}

func GetUserID(c *fiber.Ctx) (int, error) {
	if config.Config.Debug {
		return 1, nil
	}

	id, err := strconv.Atoi(c.Get("X-Consumer-Username"))
	if err != nil {
		return 0, err
	}

	return id, nil
}
