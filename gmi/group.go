package gmi

import (
	"gemigit/db"
	"net/http"

	"github.com/labstack/echo/v4"
)

func groupRedirect(c echo.Context) error {
	return c.String(http.StatusFound, "/account/groups/" +
		c.Param("group"))
}

func groupsListRedirect(c echo.Context) error {
	return c.String(http.StatusFound, "/account/groups")
}

func isGroupOwner(c echo.Context, user db.User) (int, error) {
	groupID, err := db.GetGroupID(c.Param("group"))
	if err != nil {
		return -1, c.String(http.StatusBadRequest, err.Error())
	}
	owner, err := user.IsInGroupID(groupID)
	if err != nil {
		return -1, c.String(http.StatusBadRequest, err.Error())
	}
	if !owner {
                return -1, c.String(http.StatusBadRequest,
				       "Permission denied")
	}
	return groupID, nil
}

func SetGroupDesc(c echo.Context, user db.User) error {
	query := c.QueryString()
	id, err := isGroupOwner(c, user)
	if err != nil { return err }
	err = db.SetGroupDescription(id, query)
	if err != nil { return err }
	return groupRedirect(c)
}

func DeleteGroup(c echo.Context, user db.User) error {
	name := c.QueryString()
	/*if name == "" {
		return c.String(http.StatusInput,
				   "To confirm type the group name")
	}*/
	if name != c.Param("group") {
		return groupRedirect(c)
	}
	id, err := isGroupOwner(c, user)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	err = db.DeleteGroup(id)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return groupsListRedirect(c)
}

func LeaveGroup(c echo.Context, user db.User) (error) {
	groupID, err := db.GetGroupID(c.Param("group"))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	owner, err := user.IsInGroupID(groupID)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if owner {
                return c.String(http.StatusBadRequest,
				   "You cannot leave your own group")
	}
	err = db.DeleteMember(user.ID, groupID)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return groupsListRedirect(c)
}

func RmFromGroup(c echo.Context, user db.User) (error) {
	groupID, err := isGroupOwner(c, user)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	userID, err := db.GetUserID(c.Param("user"))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if userID == user.ID {
		return c.String(http.StatusBadRequest,
			"You cannot remove yourself from your own group")
	}
	err = db.DeleteMember(userID, groupID)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return groupRedirect(c)
}

func AddToGroup(c echo.Context, user db.User) (error) {
	query := c.QueryString()
	/*if query == "" {
		return c.String(http.StatusInput, "Username")
	}*/
	group := c.Param("group")
	owner, err := user.IsInGroup(group)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if !owner {
                return c.String(http.StatusBadRequest, "Permission denied")
	}

	if err = user.AddUserToGroup(group, query); err != nil {
                return c.String(http.StatusBadRequest, err.Error())
	}
	return groupRedirect(c)
}
