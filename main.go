package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Nerzal/gocloak/v8"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

	home, err := homedir.Dir()
	if err != nil {
		log.Fatal().Err(err).Msg("Handling home dir failed.")
	}

	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigName(".keycloak")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		log.Info().Msgf("Using config file '%s'.", viper.ConfigFileUsed())
		students := viper.Get("students")

		client := gocloak.NewClient(viper.GetString("keycloak.host"))
		ctx := context.Background()
		realm := viper.GetString("keycloak.realm")
		token, err := client.LoginAdmin(ctx, viper.GetString("keycloak.user"), viper.GetString("keycloak.password"), "master")
		if err != nil {
			log.Fatal().Err(err).Msg("Something wrong with the credentials or url.")
		}

		//students := viper.Get("students")
		for _, student := range students.([]interface{}) {
			firstname := student.(map[interface{}]interface{})["firstname"].(string)
			lastname := student.(map[interface{}]interface{})["lastname"].(string)
			username := fmt.Sprintf("%c%s", firstname[0], lastname)
			// user name to lower case
			username = strings.ToLower(username)
			user := gocloak.User{
				FirstName:     gocloak.StringP(firstname),
				LastName:      gocloak.StringP(lastname),
				Email:         gocloak.StringP(student.(map[interface{}]interface{})["email"].(string)),
				Username:      gocloak.StringP(username),
				Enabled:       gocloak.BoolP(true),
				EmailVerified: gocloak.BoolP(true),
			}
			log.Info().Msgf("Creating user '%s'.", username)
			id, err := client.CreateUser(ctx, token.AccessToken, realm, user)
			if err != nil {
				log.Fatal().Err(err).Msgf("Creating user '%s' failed.", username)
				continue
			}
			actions := []string{"UPDATE_PASSWORD"}
			duration := int((7 * 24 * time.Hour).Seconds())
			params := gocloak.ExecuteActionsEmail{
				Actions:     &actions,
				Lifespan:    gocloak.IntP(duration),
				UserID:      gocloak.StringP(id),
				RedirectURI: gocloak.StringP(viper.GetString("keycloak.redirect_uri")),
				ClientID:    gocloak.StringP(viper.GetString("keycloak.client_id")),
			}
			err = client.ExecuteActionsEmail(ctx, token.AccessToken, realm, params)
			if err != nil {
				log.Fatal().Err(err).Msgf("Triggering actions email 'UPDATE_PASSWORD' for user '%s' failed.", username)

			}
		}
	} else {
		log.Fatal().Err(err).Msg("fatal error config file.")
	}
}
