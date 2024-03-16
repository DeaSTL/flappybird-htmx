package middlware

import (
	"log"
	"net/http"

	"github.com/deastl/flappybird-htmx/game"
	"github.com/deastl/flappybird-htmx/models"
	"github.com/deastl/flappybird-htmx/services"
)

func InitializeUserSession(server_state *game.ServerState) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			perm_session, err := r.Cookie("perm_jwt")

			if perm_session == nil || perm_session.Value == "" {
				err = nil
				new_user := models.User{
					Name: "",
				}
				err = services.UserCreate(server_state.Ctx, server_state.Dbq, &new_user)

				if err != nil {
					log.Printf("Error creating user %s : %v", new_user.ID, err)
				}
				perm_token, err := server_state.GeneratePermJWT(new_user.ID)

				if err != nil {
					log.Printf("Error generating JWT token %s : %v", new_user.ID, err)
				}

				perm_session = &http.Cookie{
					Name:  "perm_jwt",
					Value: perm_token,
				}
				http.SetCookie(w, perm_session)
			}

			next.ServeHTTP(w, r)

		})
	}
}
