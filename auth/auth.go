package auth

import (
	"errors"
	"wzgit/config"
	"wzgit/db"
	"wzgit/access"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var userAttempts = make(map[string]int)
var clientAttempts = make(map[string]int)
var registrationAttempts = make(map[string]int)
var loginToken = make(map[string]db.User)

func Decrease() {
	for {
		userAttempts = make(map[string]int)
		clientAttempts = make(map[string]int)
		registrationAttempts = make(map[string]int)
		loginToken = make(map[string]db.User)
		time.Sleep(time.Duration(config.Cfg.Protection.Reset) *
			   time.Second)
	}
}


func can(attemps *map[string]int, key string, max int) bool {
	value, exist := (*attemps)[key]
	return !exist || value < max
}

func try(attemps *map[string]int, key string, max int) bool {
	value, exist := (*attemps)[key]
	if exist {
		if value < max {
			(*attemps)[key]++
		} else {
			return true
		}
	} else {
		(*attemps)[key] = 1
	}
	return false
}

// Check if credential are valid then add client signature
// as a connected user
func Connect(username string, password string,
	     signature string, ip string) error {

	if try(&userAttempts, username, config.Cfg.Protection.Account) {
		return errors.New("The account is locked, " +
				  "too many connections attempts")
	}

	if try(&clientAttempts, ip, config.Cfg.Protection.Ip) {
		return errors.New("Too many connections attempts")
	}

	err := access.Login(username, password, false, true, false)
	if err != nil {
		return err
	}

	user, err := db.FetchUser(username, signature)
	if err == nil {
		if user.Secret != "" {
			loginToken[signature] = user
			return errors.New("token required")
		}
		user.CreateSession(signature)
		return nil
	}
	if !config.Cfg.Ldap.Enabled { return err }
	err = db.Register(username, "")
	if err != nil { return err }
	user, err = db.FetchUser(username, signature)
	if err != nil { return err }
	user.CreateSession(signature)
	return nil
}

func Register(username string, password string, ip string) error {
	const tooMany = "Too many registration attempts"
	if !can(&registrationAttempts, ip, config.Cfg.Protection.Registration) {
		return errors.New(tooMany)
	}
	err := db.Register(username, password)
	if err != nil { return err }
	if try(&registrationAttempts, ip, config.Cfg.Protection.Registration) {
		return errors.New(tooMany)
	}
	return nil
}

var options = totp.ValidateOpts{
	Algorithm: otp.AlgorithmSHA512,
	Period: 30,
	Digits: 6,
}

func LoginOTP(signature string, code string) error {
	user, exist := loginToken[signature]
	if !exist { return errors.New("Invalid request") }
	if err := CheckOTP(user.Secret, code); err != nil { return err }
	user.CreateSession(signature)
	delete(loginToken, signature)
	return nil
}

func CheckOTP(key string, code string) error {
	ok, err := totp.ValidateCustom(code, key, time.Now(), options)
	if err != nil { return err }
	if !ok { return errors.New("Invalid 2FA answer") }
	return nil
}

func GenerateOTP(username string) (*otp.Key, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: config.Cfg.Title,
		AccountName: username,
		Algorithm: options.Algorithm,
		Digits: options.Digits,
		Period: options.Period,
	})
	return key, err
}
