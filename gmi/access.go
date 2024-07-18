package gmi

import (
	"gemigit/db"
	"net/http"

	"github.com/labstack/echo/v4"
)

func accessRedirect(c echo.Context) error {
	return c.Redirect(http.StatusFound,
		"/account/repo/" + c.Param("repo") + "/access")
}

func privilegeUpdate(privilege int, first bool) int {
	if first {
		return (privilege + 1)%3
	}
	if privilege == 0 {
		return 2
	}
	return privilege - 1
}

func privilegeToString(privilege int) string {
	switch (privilege) {
	case 0:
		return "none"
	case 1:
		return "read"
	case 2:
		return "read and write"
	}
	return "Invalid value"
}

func accessFirstOption(privilege int) string {
	switch (privilege) {
	case 0:
                return "Grant read access"
	case 1:
                return "Grant write access"
	default:
                return "Revoke read and write access"
	}
}

func accessSecondOption(privilege int) string {
	switch (privilege) {
	case 0:
                return "Grant read and write access"
	case 1:
                return "Revoke read access"
	default:
                return "Revoke write access"
	}
}

func changeGroupAccess(user db.User, repository string,
		       name string, first bool) error {
	repo, err := user.GetRepo(repository)
	if err != nil {
		return err
	}
	groupID, err := db.GetGroupID(name)
	if err != nil {
		return err
	}
	privilege, err := db.GetGroupAccess(repo, groupID)
	if err != nil {
		return err
	}
	privilege = privilegeUpdate(privilege, first)
	err = user.SetGroupAccess(repo, groupID, privilege)
	return err
}

func groupAccessOption(c echo.Context, user db.User, first bool) error {
	err := changeGroupAccess(user, c.Param("repo"),
				 c.Param("group"), first)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return accessRedirect(c)
}

func GroupAccessFirstOption(c echo.Context, user db.User) error {
	return groupAccessOption(c, user, true)
}

func GroupAccessSecondOption(c echo.Context, user db.User) error {
	return groupAccessOption(c, user, false)
}

func changeUserAccess(owner db.User, repository string,
		      name string, first bool) error {
	repo, err := owner.GetRepo(repository)
	if err != nil {
		return err
	}
	user, err := db.GetPublicUser(name)
	if err != nil {
		return err
	}
	privilege, err := db.GetUserAccess(repo, user)
	if err != nil {
		return err
	}
	privilege = privilegeUpdate(privilege, first)
	err = owner.SetUserAccess(repo, user.ID, privilege)
	return err
}

func userAccessOption(c echo.Context, user db.User, first bool) error {
	err := changeUserAccess(user, c.Param("repo"), c.Param("user"), first)
	if err != nil {
                return c.String(http.StatusBadRequest, err.Error())
	}
	return accessRedirect(c)
}

func UserAccessFirstOption(c echo.Context, user db.User) error {
	return userAccessOption(c, user, true)
}

func UserAccessSecondOption(c echo.Context, user db.User) error {
	return userAccessOption(c, user, false)
}

func addAcess(c echo.Context, param string, user db.User) (
				string, db.User, db.Repo, error) {
	query := c.QueryString()
	/*
	if query == "" {
		return "", db.User{}, db.Repo{},
		       c.NoContent(gig.StatusInput, param)
	}
	*/
	repo, err := user.GetRepo(c.Param("repo"))
	if err != nil {
                return query, user, db.Repo{},
       			c.String(http.StatusBadRequest,
				"Repository not found")
	}
	return query, user, repo, nil
}

func AddUserAccess(c echo.Context, u db.User) error {
	query, user, repo, err := addAcess(c, "User", u)
	if err != nil {
		return err
	}
	err = user.AddUserAccess(repo, query)
	if err != nil {
                return c.String(http.StatusBadRequest, "Invalid user")
	}
	return accessRedirect(c)
}

func AddGroupAccess(c echo.Context, u db.User) error {
	query, user, repo, err := addAcess(c, "Group", u)
	if err != nil {
		return err
	}
	err = user.AddGroupAccess(repo, query)
	if err != nil {
                return c.String(http.StatusBadRequest, "Invalid user")
	}
	return accessRedirect(c)
}

func RemoveUserAccess(c echo.Context, user db.User) error {
	userID, err := db.GetUserID(c.Param("user"))
	if err != nil {
                return c.String(http.StatusBadRequest, "User not found")
	}
	repo, err := user.GetRepo(c.Param("repo"))
	err = user.RemoveUserAccess(repo, userID)
	if err != nil {
                return c.String(http.StatusBadRequest,
				"User doesn't have access")
	}
	return accessRedirect(c)
}

func RemoveGroupAccess(c echo.Context, user db.User) error {
	groupID, err := db.GetGroupID(c.Param("group"))
	if err != nil {
                return c.String(http.StatusBadRequest, "Group not found")
	}
	repo, err := user.GetRepo(c.Param("repo"))
	err = user.RemoveGroupAccess(repo, groupID)
	if err != nil {
                return c.String(http.StatusBadRequest,
                                   "Group doesn't have access")
	}
	return accessRedirect(c)
}
