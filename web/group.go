package web 

import (
	"wzgit/db"
	"errors"

	"github.com/labstack/echo/v4"
)

func groupRedirect(c echo.Context, after string) error {
	return redirection(c, "/account/groups", after)
}

func isGroupOwner(c echo.Context, user db.User) (int, error) {
	groupID, err := db.GetGroupID(c.Param("group"))
	if err != nil { return -1, err }
	owner, err := user.IsInGroupID(groupID)
	if err != nil { return -1, err }
	if !owner { return -1, errors.New("Permission denied") }
	return groupID, nil
}

func SetGroupDesc(c echo.Context, user db.User) error {
	desc := c.Request().PostFormValue("description")
	id, err := isGroupOwner(c, user)
	if err != nil { return err }
	err = db.SetGroupDescription(id, desc)
	if err != nil { return err }
	return groupRedirect(c, c.Param("group"))
}

func AddGroup(c echo.Context, user db.User) error {
	name := c.Request().PostFormValue("group")
	if err := user.CreateGroup(name); err != nil { return err }
	return groupRedirect(c, name)
}

func DeleteGroup(c echo.Context, user db.User) error {
	name := c.QueryString()
	if name != c.Param("group") {
		user.Set("group_delete_confirm", c.Param("group"))
		return groupRedirect(c, "")
	}
	id, err := isGroupOwner(c, user)
	if err != nil { return err }
	err = db.DeleteGroup(id)
	if err != nil { return err }
	return groupRedirect(c, "")
}

func LeaveGroup(c echo.Context, user db.User) (error) {
	groupID, err := db.GetGroupID(c.Param("group"))
	if err != nil { return err }
	owner, err := user.IsInGroupID(groupID)
	if err != nil { return err }
	if owner { return errors.New("You cannot leave your own group") }
	err = db.DeleteMember(user.ID, groupID)
	if err != nil { return err }
	return groupRedirect(c, "")
}

func RmFromGroup(c echo.Context, user db.User) (error) {
	groupID, err := isGroupOwner(c, user)
	if err != nil { return err }
	userID, err := db.GetUserID(c.Param("user"))
	if err != nil { return err }
	if userID == user.ID {
		return errors.New(
			"You cannot remove yourself from your own group")
	}
	err = db.DeleteMember(userID, groupID)
	if err != nil { return err }
	return groupRedirect(c, c.Param("group"))
}

func AddToGroup(c echo.Context, user db.User) (error) {
	query := c.Request().PostFormValue("name")
	group := c.Param("group")
	owner, err := user.IsInGroup(group)
	if err != nil { return err }
	if !owner { return errors.New("Permission denied") }
	if err = user.AddUserToGroup(group, query); err != nil { return err }
	return groupRedirect(c, group)
}
