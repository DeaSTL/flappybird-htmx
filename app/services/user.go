package services

import (
	"context"
	"log"
	"time"

	"github.com/deastl/flappybird-htmx/db"
	"github.com/deastl/flappybird-htmx/models"
	"github.com/deastl/flappybird-htmx/utils"
)

func userFromDb(db_user *db.User, model_user *models.User) error {
	created_at, err := time.Parse(time.RFC3339, db_user.CreatedAt)
	if err != nil {
		return err
	}
	updated_at, err := time.Parse(time.RFC3339, db_user.UpdatedAt)
	if err != nil {
		return err
	}

	model_user.CreatedAt = created_at
	model_user.UpdatedAt = updated_at

	model_user.Name = db_user.Name
	model_user.TopScore = int(db_user.TopScore)
	model_user.LastScore = int(db_user.LastScore)

	return nil
}

func UserCreate(ctx context.Context, q *db.Queries, new_user *models.User) error {

	new_user.ID = utils.GenID(32)
	new_user.CreatedAt = time.Now()
	new_user.UpdatedAt = time.Now()
	new_user.TopScore = 0
	new_user.LastScore = 0
	db_user_params := db.CreateUserParams{
		Name:      new_user.Name,
		CreatedAt: new_user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: new_user.UpdatedAt.Format(time.RFC3339),
		ID:        new_user.ID,
		TopScore:  int64(new_user.TopScore),
		LastScore: int64(new_user.LastScore),
	}

	return q.CreateUser(ctx, db_user_params)
}

func UserGetByName(ctx context.Context, q *db.Queries, name string) (models.User, error) {
	db_user, err := q.GetUserByName(ctx, name)
	if err != nil {
		log.Fatalf("error in UserGetByName : %v", err)
	}

	model_user := models.User{}
	err = userFromDb(&db_user, &model_user)

	if err != nil {
		return models.User{}, err
	}

	return model_user, nil
}
