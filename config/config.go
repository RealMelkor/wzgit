package config

import "github.com/kkyr/fig"

var Cfg Config

type Config struct {
	Title		string `validate:"required"`
	Database struct {
		Type		string `validate:"required"`
		Url		string `validate:"required"`
	}
	Web struct {
		Domain		string `default:"localhost"`
		Host		string `default:":8080"`
	}
	Catpcha struct {
		Enabled		bool
		Length		int
	}
	Git struct {
		Http struct {
			Enabled		bool
			Https		bool
			Domain		string	`validate:"required"`
		}
		SSH struct {
			Enabled		bool
			Domain		string	`validate:"required"`
			Address 	string  `validate:"required"`
			Port		int	`validate:"required"`
			Key		string	`validate:"required"`
		}
		Path		string	`validate:"required"`
		Public		bool
		MaximumCommits	int
	}
	Ldap struct {
		Enabled		bool
		Url		string
		Attribute	string
		Binding		string
	}
	Users struct {
		Registration	bool
	}
	Protection struct {
		Ip		int    `validate:"required"`
		Account		int    `validate:"required"`
		Registration	int    `validate:"required"`
		Reset		int    `validate:"required"`
	}
}

func LoadConfig() error {
	err := fig.Load(
		&Cfg,
		fig.File("config.yaml"),
		fig.Dirs(".", "/etc/wzgit", "/usr/local/etc/wzgit"),
	)
	if err == nil && Cfg.Ldap.Enabled {
		Cfg.Users.Registration = false
	}
	return err
}
