package main

import (
	"context"
	"time"

	"github.com/pdcgo/shared/authorization"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/shared/interfaces/identity_iface"
	"gorm.io/gorm"
)

type Helper struct {
	CreateTokenFromUsername CreateTokenFromUsername
	SetAuthorization        SetAuthorization
}

func NewHelper(
	CreateTokenFromUsername CreateTokenFromUsername,
	SetAuthorization SetAuthorization,
) *Helper {
	return &Helper{
		CreateTokenFromUsername,
		SetAuthorization,
	}
}

type CreateTokenFromUsername func(username string) (string, error)

func NewCreateTokenFromUsername(
	db *gorm.DB,
	cfg *configs.AppConfig,
) CreateTokenFromUsername {
	return func(username string) (string, error) {
		var user db_models.User
		err := db.Model(&db_models.User{}).Where("username = ?", username).First(&user).Error
		if err != nil {
			return "", err
		}
		jwt := authorization.JwtIdentity{
			UserID:     user.ID,
			SuperUser:  user.IsSuperUser(),
			UserAgent:  identity_iface.TestAgent,
			CreatedAt:  time.Now().UnixMicro(),
			ValidUntil: time.Now().Add(time.Hour * 24 * 7).UnixMicro(),
		}

		return jwt.Serialize(cfg.JwtSecret)
	}
}

type SetAuthorization func(ctx context.Context, username string) (context.Context, error)

func NewSetAuthorization(
	CreateTokenFromUsername CreateTokenFromUsername,
	cfg *configs.AppConfig,
) SetAuthorization {
	return func(ctx context.Context, username string) (context.Context, error) {
		token, err := CreateTokenFromUsername(username)
		if err != nil {
			return ctx, err
		}

		ctx = custom_connect.SetAuthToken(ctx, token)
		return ctx, nil
	}
}
