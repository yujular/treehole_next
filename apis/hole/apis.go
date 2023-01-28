package hole

import (
	"fmt"
	"strconv"
	. "treehole_next/models"
	. "treehole_next/utils"
	"treehole_next/utils/perm"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ListHolesByDivision
//
//	@Summary	List Holes In A Division
//	@Tags		Hole
//	@Produce	json
//	@Router		/divisions/{division_id}/holes [get]
//	@Param		division_id	path		int			true	"division_id"
//	@Param		object		query		QueryTime	false	"query"
//	@Success	200			{array}		Hole
//	@Failure	404			{object}	MessageModel
//	@Failure	500			{object}	MessageModel
func ListHolesByDivision(c *fiber.Ctx) error {
	var query QueryTime
	err := ValidateQuery(c, &query)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return err
	}

	// get holes
	var holes Holes
	querySet, err := holes.MakeQuerySet(query.Offset, query.Size, "", c)
	if err != nil {
		return err
	}
	if id != 0 {
		querySet = querySet.Where("division_id = ?", id)
	}
	querySet.Find(&holes)

	return Serialize(c, &holes)
}

// ListHolesByTag
//
//	@Summary	List Holes By Tag
//	@Tags		Hole
//	@Produce	json
//	@Router		/tags/{tag_name}/holes [get]
//	@Param		tag_name	path		string		true	"tag_name"
//	@Param		object		query		QueryTime	false	"query"
//	@Success	200			{array}		Hole
//	@Failure	404			{object}	MessageModel
func ListHolesByTag(c *fiber.Ctx) error {
	var query QueryTime
	err := ValidateQuery(c, &query)
	if err != nil {
		return err
	}

	// get tag
	var tag Tag
	tagName := c.Params("name")
	result := DB.Where("name = ?", tagName).First(&tag)
	if result.Error != nil {
		return result.Error
	}

	// get holes
	var holes Holes
	querySet, err := holes.MakeQuerySet(query.Offset, query.Size, "", c)
	if err != nil {
		return err
	}
	err = querySet.Model(&tag).
		Association("Holes").Find(&holes)
	if err != nil {
		return err
	}

	return Serialize(c, &holes)
}

// ListHolesOld
//
//	@Summary	Old API for Listing Holes
//	@Deprecated
//	@Tags		Hole
//	@Produce	application/json
//	@Router		/holes [get]
//	@Param		object	query	ListOldModel	false	"query"
//	@Success	200		{array}	Hole
func ListHolesOld(c *fiber.Ctx) error {
	var query ListOldModel
	err := ValidateQuery(c, &query)
	if err != nil {
		return err
	}

	var holes Holes
	querySet, err := holes.MakeQuerySet(query.Offset, query.Size, query.Order, c)
	if err != nil {
		return err
	}
	if query.Tag != "" {
		var tag Tag
		result := DB.Where("name = ?", query.Tag).First(&tag)
		if result.Error != nil {
			return result.Error
		}
		err = querySet.Model(&tag).Association("Holes").Find(&holes)
		if err != nil {
			return err
		}
	} else if query.DivisionID != 0 {
		querySet.
			Where("division_id = ?", query.DivisionID).
			Find(&holes)
	} else {
		querySet.Find(&holes)
	}

	// only for danxi v1.3.10 old api
	if query.Order == "time_created" || query.Order == "created_at" {
		for i := range holes {
			holes[i].UpdatedAt = holes[i].CreatedAt
		}
	}

	return Serialize(c, &holes)
}

// GetHole
//
//	@Summary	Get A Hole
//	@Tags		Hole
//	@Produce	application/json
//	@Router		/holes/{id} [get]
//	@Param		id	path		int	true	"id"
//	@Success	200	{object}	Hole
//	@Failure	404	{object}	MessageModel
func GetHole(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id")

	// get hole
	hole := Hole{}
	querySet, err := MakeQuerySet(c)
	if err != nil {
		return err
	}
	result := querySet.First(&hole, id)
	if result.Error != nil {
		return result.Error
	}

	return Serialize(c, &hole)
}

// CreateHole
//
//	@Summary		Create A Hole
//	@Description	Create a hole, create tags and floor binding to it and set the name mapping
//	@Tags			Hole
//	@Produce		application/json
//	@Router			/divisions/{division_id}/holes [post]
//	@Param			division_id	path		int			true	"division id"
//	@Param			json		body		CreateModel	true	"json"
//	@Success		201			{object}	Hole
func CreateHole(c *fiber.Ctx) error {
	// validate body
	var body CreateModel
	err := ValidateBody(c, &body)
	if err != nil {
		return err
	}
	divisionID, err := c.ParamsInt("id")
	if err != nil {
		return err
	}

	// todo: check permission
	// get user from auth
	user, err := GetUserFromAuth(c)
	if err != nil {
		return err
	}

	// permission
	if user.BanDivision[divisionID] {
		return Forbidden("您没有权限在此板块发言")
	}

	hole := Hole{
		Tags:       body.ToTags(),
		Floors:     Floors{{UserID: user.ID, Content: body.Content, SpecialTag: body.SpecialTag, IsMe: true}},
		UserID:     user.ID,
		DivisionID: divisionID,
	}
	err = hole.Create(DB)
	if err != nil {
		return err
	}

	return c.Status(201).JSON(&hole)
}

// CreateHoleOld
//
//	@Summary	Old API for Creating A Hole
//	@Deprecated
//	@Tags		Hole
//	@Produce	application/json
//	@Router		/holes [post]
//	@Param		json	body		CreateOldModel	true	"json"
//	@Success	201		{object}	CreateOldResponse
func CreateHoleOld(c *fiber.Ctx) error {
	// validate body
	var body CreateOldModel
	err := ValidateBody(c, &body)
	if err != nil {
		return err
	}

	// todo: check permission
	// get user from auth
	user, err := GetUserFromAuth(c)
	if err != nil {
		return err
	}

	// permission
	if user.BanDivision[body.DivisionID] {
		return Forbidden("您没有权限在此板块发言")
	}

	// create hole
	hole := Hole{
		Tags:       body.ToTags(),
		Floors:     Floors{{UserID: user.ID, Content: body.Content, SpecialTag: body.SpecialTag, IsMe: true}},
		UserID:     user.ID,
		DivisionID: body.DivisionID,
	}
	err = hole.Create(DB)
	if err != nil {
		return err
	}

	return c.Status(201).JSON(&CreateOldResponse{
		Data:    hole,
		Message: "发表成功",
	})
}

// ModifyHole
//
//	@Summary	Modify A Hole
//	@Tags		Hole
//	@Produce	application/json
//	@Router		/holes/{id} [put]
//	@Param		id		path		int			true	"id"
//	@Param		json	body		ModifyModel	true	"json"
//	@Success	200		{object}	Hole
//	@Failure	404		{object}	MessageModel
func ModifyHole(c *fiber.Ctx) error {
	// validate
	var body ModifyModel
	err := ValidateBody(c, &body)
	if err != nil {
		return err
	}

	holeID, err := c.ParamsInt("id")
	if err != nil {
		return err
	}

	// get user
	user, err := GetUser(c)
	if err != nil {
		return err
	}

	// Find hole
	var hole Hole
	result := DB.First(&hole, holeID)
	if result.Error != nil {
		return result.Error
	}

	// modify division
	if body.DivisionID != 0 && body.DivisionID != hole.DivisionID {
		if !perm.CheckPermission(user, perm.Admin) {
			return Forbidden("非管理员禁止修改分区")
		}
		hole.DivisionID = body.DivisionID
		// log
		MyLog("Hole", "Modify", holeID, user.ID, RoleAdmin, "DivisionID to: ", strconv.Itoa(hole.DivisionID))
	}

	// modify hidden
	if body.Unhidden && hole.Hidden {
		if !perm.CheckPermission(user, perm.Admin) {
			return Forbidden("非管理员禁止取消隐藏")
		}
		hole.Hidden = false
		// log
		MyLog("Hole", "Modify", holeID, user.ID, RoleAdmin, "Unhidden: ")
	}

	// modify tags
	if len(body.Tags) != 0 {
		if perm.CheckPermission(user, perm.Admin) || user.ID == hole.UserID {
			for _, tag := range body.Tags {
				hole.Tags = append(hole.Tags, &Tag{Name: tag.Name})
			}
			err = DB.Transaction(func(tx *gorm.DB) error {
				return hole.SetTags(tx, true)
			})
			if err != nil {
				return err
			}

			// log
			if perm.CheckPermission(user, perm.Admin) {
				MyLog("Hole", "Modify", holeID, user.ID, RoleAdmin, "NewTags: ", fmt.Sprintf("%v", body.Tags))
			} else {
				MyLog("Hole", "Modify", holeID, user.ID, RoleOwner, "NewTags: ", fmt.Sprintf("%v", body.Tags))
			}

		} else {
			return Forbidden()
		}
	}

	// save
	DB.Omit("Tags").Save(&hole)

	// update cache
	updateHoles := []*Hole{&hole}
	err = UpdateHoleCache(updateHoles)
	if err != nil {
		return err
	}

	return Serialize(c, &hole)
}

// DeleteHole
//
//	@Summary		Delete A Hole
//	@Description	Hide a hole, but visible to admins. This may affect many floors, DO NOT ABUSE!!!
//	@Tags			Hole
//	@Produce		application/json
//	@Router			/holes/{id} [delete]
//	@Param			id	path	int	true	"id"
//	@Success		204
//	@Failure		404	{object}	MessageModel
func DeleteHole(c *fiber.Ctx) error {
	// validate holeID
	holeID, err := c.ParamsInt("id")
	if err != nil {
		return err
	}

	// get user
	user, err := GetUser(c)
	if err != nil {
		return err
	}

	// permission
	if !perm.CheckPermission(user, perm.Admin) {
		return Forbidden()
	}

	var hole Hole
	hole.ID = holeID
	result := DB.Model(&hole).Select("Hidden").Updates(Hole{Hidden: true})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	// log
	MyLog("Hole", "Delete", holeID, user.ID, RoleAdmin)

	// find hole and update cache
	result = DB.First(&hole, holeID)
	if result.Error != nil {
		return result.Error
	}

	updateHoles := []*Hole{&hole}
	err = UpdateHoleCache(updateHoles)
	if err != nil {
		return err
	}

	return c.Status(204).JSON(nil)
}

// PatchHole
//
//	@Summary		Patch A Hole
//	@Description	Add hole.view
//	@Tags			Hole
//	@Produce		application/json
//	@Router			/holes/{id} [patch]
//	@Param			id	path	int	true	"id"
//	@Success		204
//	@Failure		404	{object}	MessageModel
func PatchHole(c *fiber.Ctx) error {
	holeID, err := c.ParamsInt("id")
	if err != nil {
		return err
	}

	holeViewsChan <- holeID

	return c.Status(204).JSON(nil)
}
