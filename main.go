package main

import (
	"fmt"
	"log"
	"os"
	"errors"

	"golang.org/x/term"

	"wzgit/access"
	"wzgit/auth"
	"wzgit/config"
	"wzgit/db"
	"wzgit/sshgit"
	"wzgit/repo"
	"wzgit/web"
)

func start() error {
	if err := config.LoadConfig(); err != nil { return err }

	if len(os.Args) > 1 {
		switch (os.Args[1]) {
		case "chpasswd":
			if (config.Cfg.Ldap.Enabled) {
				return errors.New(
					"Not valid when LDAP is enabled")
			}
			if len(os.Args) < 3 {
				return errors.New(os.Args[0] +
					    " chpasswd <username>")
			}
			fmt.Print("New Password : ")
			password, err := term.ReadPassword(0)
			fmt.Print("\n")
			if err != nil { return err }
			err = db.Init(config.Cfg.Database.Type,
				      config.Cfg.Database.Url, false)
			if err != nil { return err }
			defer db.Close()
			err = db.ChangePassword(os.Args[2], string(password))
			if err != nil { return err }
			fmt.Println(os.Args[2] + "'s password changed")
			return nil
		case "register":
			if (config.Cfg.Ldap.Enabled) {
				return errors.New(
					"Not valid when LDAP is enabled")
			}
			if len(os.Args) < 3 {
				return errors.New(os.Args[0] +
					    " register <username>")
			}
			fmt.Print("Password : ")
			password, err := term.ReadPassword(0)
			fmt.Print("\n")
			if err != nil { return err
			}
			err = db.Init(config.Cfg.Database.Type,
				      config.Cfg.Database.Url, false)
			if err != nil { return err }
			defer db.Close()
			err = db.Register(os.Args[2], string(password))
			if err != nil { return err }
			fmt.Println("User " + os.Args[2] + " created")
			return nil
		case "rmuser":
			if len(os.Args) < 3 {
				return errors.New(
					os.Args[0] + " rmuser <username>")
			}
			err := db.Init(config.Cfg.Database.Type,
				       config.Cfg.Database.Url, false)
			if err != nil { return err }
			defer db.Close()
			err = db.DeleteUser(os.Args[2])
			if err != nil { return err }
			fmt.Println("User " + os.Args[2] +
				    " deleted successfully")
			return nil
		case "init":
			err := db.Init(config.Cfg.Database.Type,
				       config.Cfg.Database.Url, true)
			if err != nil { return err }
			db.Close()
			return nil
		case "update":
			err := db.Init(config.Cfg.Database.Type,
				       config.Cfg.Database.Url, false)
			if err != nil { return err }
			db.UpdateTable()
			db.Close()
			return nil
		}
		fmt.Println("usage: " + os.Args[0] + " [command]")
		fmt.Println("commands :")
		fmt.Println("\tchpasswd <username> - Change user password")
		fmt.Println("\tregister <username> - Create user")
		fmt.Println("\trmuser <username> - Remove user")
		fmt.Println("\tupdate - Update database " +
			"(Warning, it is recommended to do a backup of " +
			"the database before using this command)")
		fmt.Println("\tinit - Initialize database")
		return nil
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := access.Init(); err != nil { return err }
	if err := web.LoadTemplate(); err != nil { return err }

	err := db.Init(config.Cfg.Database.Type,
		       config.Cfg.Database.Url, false)
	if err != nil { return err }
	defer db.Close()
	if err := repo.Init("repos"); err != nil { return err }

	if config.Cfg.Git.SSH.Enabled {
		go sshgit.Listen(config.Cfg.Git.Path,
			config.Cfg.Git.SSH.Address,
			config.Cfg.Git.SSH.Port)
	}
	go auth.Decrease()

	return web.Listen()
}

func main() {
	if err := start(); err != nil {
		log.Println(err)
	}
}
