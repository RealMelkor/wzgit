package csrf

import (
	"errors"
	"crypto/rand"
	"gemigit/db"

	"github.com/labstack/echo/v4"
)

var tokens = map[string]string{}

func getUser(c echo.Context) (db.User, error) {
	cookie, err := c.Cookie("auth_id")
	if err != nil {
		return db.User{}, err
	}
	user, exist := db.GetUser(cookie.Value)
	if !exist {
		return db.User{}, errors.New("user not found")
	}
	return user, nil
}

const characters = "abcdefghijklmnopqrstuvwxyz0123456789"
func randomString(n int) string {
        var random [1024]byte
	if n > 1024 { return "" }
        b := make([]byte, n)
        rand.Read(random[:n])
        for i := range b {
                b[i] = characters[int64(random[i]) % int64(len(characters))]
        }
        return string(b)
}

func New(user db.User) error {
	token := randomString(16)
	tokens[user.Signature] = token
	return nil
}

func Verify(user db.User, csrf string) error {
	token, exist := tokens[user.Signature]
	if !exist || token != csrf {
		return errors.New("invalid csrf token")
	}
	return nil
}

func Handle(user db.User, token string) error {
	if token == "" {
		New(user)
	} else if err := Verify(user, token); err != nil {
		return err
	}
	return nil
}

func Token(sig string) string {
	return tokens[sig]
}
