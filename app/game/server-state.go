package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/deastl/flappybird-htmx/db"
	"github.com/deastl/flappybird-htmx/metrics"
	"github.com/deastl/flappybird-htmx/utils"
	"github.com/golang-jwt/jwt"
)

type ServerState struct {
	GameStates    sync.Map
	Templates     *template.Template
	JWTSecret     string
	Dbq           *db.Queries
	Ctx           context.Context
	Mut           sync.Mutex
	frames_served int
	last_sample_frames int // Track frames since last sample
}

func (s *ServerState) New() {
	s.Templates = template.New("").Funcs(template.FuncMap{
		"add": func(i, j int) int {
			return i + j
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
	})
	default_secret := "this_is_a_fake_secret"
	env_secret := os.Getenv("jwt_secret")

	if len(s.JWTSecret) == 0 {
		s.JWTSecret = env_secret
		if len(s.JWTSecret) == 0 {
			s.JWTSecret = default_secret
		}
	}

	s.initTempaltes()
}

func (s *ServerState) initTempaltes() {

	x := []string{
		"templates/bounding-box.tmpl.css",
		"templates/index.tmpl.html",
		"templates/game-wrapper.tmpl.html",
		"templates/pipe.tmpl.css",
		"templates/player.tmpl.css",
		"templates/screen.tmpl.html",
		"templates/screen-frame.tmpl.html",
		"templates/stats.tmpl.html",
		"templates/ui-overlay.tmpl.html",
		"templates/leaderboard.tmpl.html",
	}
	for _, f := range x {
		fileContents, err := os.ReadFile(f)
		if err != nil {
			panic(err.Error())
		}
		_, err = s.Templates.New(f).Parse(utils.MinifyTemplate(string(fileContents)))
		if err != nil {
			panic(err.Error())
		}

	}
}

func (s *ServerState) LogInfo() {
	var activePlayerCount float64
	var connectedSessionCount float64
	var totalFPS float64
	var activeSessions int

	s.GameStates.Range(func(key any, value any) bool {
		id := key.(string)
		game_state := value.(*GameState)
		
		connectedSessionCount++
		
		playerAlive := !game_state.Player.Dead
		

		// Update per-session metrics
		metrics.SessionScore.WithLabelValues(id).Set(float64(game_state.Points))
		metrics.SessionTargetFPS.WithLabelValues(id).Set(float64(game_state.TargetFPS))
		
		if playerAlive {
			activePlayerCount++
			activeSessions++
			totalFPS += float64(game_state.FPS)
			metrics.SessionFPS.WithLabelValues(id).Set(float64(game_state.FPS))
		} else {
			// Clean up FPS metric for dead players but keep score/target FPS
			metrics.SessionFPS.DeleteLabelValues(id)
		}
		
		return true
	})

	// Update aggregate metrics
	metrics.ActivePlayers.Set(activePlayerCount)
	metrics.ConnectedSessions.Set(connectedSessionCount)
	
	// Update average FPS
	if activeSessions > 0 {
		metrics.AverageFPS.Set(totalFPS / float64(activeSessions))
	} else {
		metrics.AverageFPS.Set(0)
	}

	// Update frame metrics
	framesSinceLastSample := s.frames_served - s.last_sample_frames
	metrics.FramesServedLastSample.Set(float64(framesSinceLastSample))
	s.last_sample_frames = s.frames_served
}

func (s *ServerState) NewPhysicsSession(session_id string) {
	go func(session_id string, server_state *ServerState) {
		delay := 30 * time.Millisecond
		for {
			sync_out, ok := s.GameStates.Load(session_id)

			game_state := sync_out.(*GameState)

			if !game_state.ClientAlive {
				return
			}

			if !ok {
				log.Print("Could not load game state in game loop")
				return
			}
			game_state.Player.Update()
			game_state.Update()
			server_state.GameStates.Store(session_id, game_state)

			time.Sleep(time.Duration(delay))
		}
	}(session_id, s)
}

func (s *ServerState) PlayerRequestedFrame(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/html")

	// Increment frame counter and metrics
	s.frames_served++
	metrics.TotalFramesServed.Inc()

	game_state, err := s.GetSessionGameState(r)

	if game_state.FrameTimer == nil {
		game_state.FrameTimer = time.NewTimer(3 * time.Second)
	}

	select {
	case <-game_state.FrameTimer.C:
		game_state.FPS = game_state.FrameCount / 3
		game_state.FrameCount = 0
		game_state.FrameTimer.Reset(3 * time.Second)
	default:
		game_state.FrameCount++
	}

	game_state.TotalFrameCount++

	if game_state.Player.Dead && game_state.DeadScreenTimer == nil {
		game_state.DeadScreenTimer = time.NewTimer(10 * time.Second)
		
		// Save score to database when player first dies
		if !game_state.ScoreSaved {
			s.savePlayerScore(r, game_state)
			game_state.ScoreSaved = true
		}
	}

	if game_state.DeadScreenTimer != nil {
		select {
		case <-game_state.DeadScreenTimer.C:
			w.Header().Set("Hx-Trigger", "get-dead-screen")
			game_state.ClientAlive = false
		default:
		}
	}

	if err != nil {
		return errors.New("Error in get-screen: " + err.Error())
	}

	game_state.Mut.Lock()
	err = s.Templates.ExecuteTemplate(w, "templates/screen.tmpl.html", game_state)
	game_state.Mut.Unlock()

	if err != nil {
		return errors.New("Could not render index template: " + err.Error())
	}
	return nil
}
func (s *ServerState) PlayerJumped(w http.ResponseWriter, r *http.Request) error {
	game_state, err := s.GetSessionGameState(r)

	if err != nil {
		log.Printf("Error in jump-player: %v", err)
		return err
	}

	// Check if this is the first jump
	firstJump := !game_state.Player.Started

	game_state.Player.Started = true
	game_state.Player.Jumping = true

	s.SetSessionGameState(r, game_state)

	// If this is the first jump, trigger polling to start
	if firstJump {
		w.Header().Set("HX-Trigger", "start-polling")
	}

	w.WriteHeader(200)

	return nil
}

func (s *ServerState) InitializePlayerSession(w http.ResponseWriter, r *http.Request) (string, error) {

	temp_session := &http.Cookie{
		Name:  "session",
		Value: utils.GenID(32),
	}

	log.Printf("New user from: %s", temp_session.Value)

	http.SetCookie(w, temp_session)

	return temp_session.Value, nil
}

func (s *ServerState) PlayerEntered(w http.ResponseWriter, r *http.Request) error {

	temp_session_id, err := s.InitializePlayerSession(w, r)

	if err != nil {
		return errors.New("Could not initalize user session")
	}

	new_game_state := NewGameState()

	s.GameStates.Store(temp_session_id, new_game_state)

	s.NewPhysicsSession(temp_session_id)

	err = s.Templates.ExecuteTemplate(w, "templates/index.tmpl.html", new_game_state)
	if err != nil {
		return err
	}

	return nil
}

func (s *ServerState) GetSessionGameState(r *http.Request) (*GameState, error) {
	s.Mut.Lock()
	defer s.Mut.Unlock()
	session, err := r.Cookie("session")

	session_id := session.Value

	if err != nil {
		return &GameState{}, err
	}

	sync_state, ok := s.GameStates.Load(session_id)
	if !ok {
		return &GameState{}, errors.New("coud not load sync state from syncmap")
	}

	game_state := sync_state.(*GameState)

	return game_state, nil
}

func (s *ServerState) SetSessionGameState(r *http.Request, game_state *GameState) error {

	session_id, err := r.Cookie("session")

	if err != nil {
		return err
	}

	s.GameStates.Store(session_id.Value, game_state)

	return nil
}

// func (s *ServerState) getUser(r *http.Request) (models.User, error) {
//
// }

type Claims struct {
	UserID string `json:"user_id"`
	jwt.StandardClaims
}

func (s *ServerState) ParsePermJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.JWTSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("Invalid token")
	}

	return claims, nil
}

func (s *ServerState) GeneratePermJWT(user_id string) (string, error) {

	expire_time := time.Now().Add(time.Hour * 100000)

	claims := &Claims{
		UserID: user_id,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expire_time.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	token_string, err := token.SignedString(s.JWTSecret)

	if err != nil {
		return "", err
	}

	return token_string, nil
}

func (s *ServerState) savePlayerScore(r *http.Request, game_state *GameState) {
	// Get username from cookie
	usernameCookie, err := r.Cookie("username")
	if err != nil {
		log.Printf("No username cookie found when saving score: %v", err)
		return
	}
	
	username := usernameCookie.Value
	score := int64(game_state.Points)
	now := time.Now().Format(time.RFC3339)
	
	// Try to get existing user
	existingUser, err := s.Dbq.GetUserByName(s.Ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create new user
			userId := utils.GenID(16)
			err = s.Dbq.CreateUser(s.Ctx, db.CreateUserParams{
				ID:        userId,
				Name:      username,
				CreatedAt: now,
				UpdatedAt: now,
				LastScore: score,
				TopScore:  score,
			})
			if err != nil {
				log.Printf("Error creating new user: %v", err)
			}
		} else {
			log.Printf("Error getting user by name: %v", err)
		}
		return
	}
	
	// Update existing user
	newTopScore := existingUser.TopScore
	if score > existingUser.TopScore {
		newTopScore = score
	}
	
	err = s.Dbq.UpdateUserScore(s.Ctx, db.UpdateUserScoreParams{
		LastScore: score,
		TopScore:  newTopScore,
		UpdatedAt: now,
		Name:      username,
	})
	if err != nil {
		log.Printf("Error updating user score: %v", err)
	}
}
