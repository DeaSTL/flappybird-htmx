package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deastl/flappybird-htmx/db"
	"github.com/deastl/flappybird-htmx/game"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// FiberResponseWriter wraps fiber.Ctx to implement http.ResponseWriter
type FiberResponseWriter struct {
	ctx    *fiber.Ctx
	buf    bytes.Buffer
	header http.Header
}

func (w *FiberResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *FiberResponseWriter) Write(data []byte) (int, error) {
	// Apply headers to Fiber context when first writing
	w.applyHeaders()
	return w.buf.Write(data)
}

func (w *FiberResponseWriter) WriteHeader(statusCode int) {
	// Apply headers when WriteHeader is called (in case Write is not called)
	w.applyHeaders()
	w.ctx.Status(statusCode)
}

func (w *FiberResponseWriter) applyHeaders() {
	if w.header != nil {
		for key, values := range w.header {
			if key == "Set-Cookie" {
				// Handle cookies specially
				for _, cookieStr := range values {
					w.ctx.Response().Header.Add("Set-Cookie", cookieStr)
				}
			} else {
				for _, value := range values {
					w.ctx.Set(key, value)
				}
			}
		}
		w.header = nil // Clear headers after setting them
	}
}

// convertFiberToHTTP converts a Fiber context to http.Request
func convertFiberToHTTP(c *fiber.Ctx) *http.Request {
	req := &http.Request{
		Method: c.Method(),
		URL: &url.URL{
			Scheme:   c.Protocol(),
			Host:     string(c.Request().Host()),
			Path:     c.Path(),
			RawQuery: string(c.Request().URI().QueryString()),
		},
		Header: make(http.Header),
	}
	
	// Convert headers including cookies
	c.Request().Header.VisitAll(func(key, value []byte) {
		req.Header.Add(string(key), string(value))
	})
	
	// Add form parsing method
	if c.Method() == "POST" && strings.Contains(c.Get("Content-Type"), "application/x-www-form-urlencoded") {
		req.Form = make(url.Values)
		c.Request().PostArgs().VisitAll(func(key, value []byte) {
			req.Form.Add(string(key), string(value))
		})
	}
	
	return req
}

func generateGuestName() string {
	adjectives := []string{"Flying", "Speedy", "Jumping", "Soaring", "Swift", "Agile", "Brave", "Daring", "Epic", "Mighty"}
	nouns := []string{"Bird", "Flyer", "Eagle", "Hawk", "Pilot", "Ace", "Hero", "Master", "Champion", "Legend"}
	
	randomAdj := adjectives[rand.Intn(len(adjectives))]
	randomNoun := nouns[rand.Intn(len(nouns))]
	randomNum := rand.Intn(100)
	
	return fmt.Sprintf("%s%s%d", randomAdj, randomNoun, randomNum)
}

func main() {

	server_state := game.ServerState{}

	dbq, err := db.NewConnection()

	if err != nil {
		log.Fatalf("Could not open database connection : %v", err)
	}

	ctx := context.Background()

	server_state.Dbq = dbq
	server_state.Ctx = ctx

	server_state.New()

	log.Println("Starting flappybird server")

	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
	})

	// Add middleware
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	
	// Only enable logging middleware in development (not in production)
	if os.Getenv("ENVIRONMENT") != "production" {
		app.Use(logger.New(logger.Config{
			Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
		}))
	}

	// Serve static files
	app.Static("/local", "./local")

	// Report info to log
	go func() {
		for {
			time.Sleep(5 * time.Second)
			server_state.LogInfo()
		}
	}()

	app.Get("/", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return server_state.Templates.ExecuteTemplate(c.Response().BodyWriter(), "templates/game-wrapper.tmpl.html", nil)
	})

	app.Get("/game", func(c *fiber.Ctx) error {
		w := &FiberResponseWriter{ctx: c}
		r := convertFiberToHTTP(c)
		
		err := server_state.PlayerEntered(w, r)
		if err != nil {
			return c.Status(500).SendString("Error when initializing player: " + err.Error())
		}
		
		c.Set("Content-Type", "text/html")
		return c.Send(w.buf.Bytes())
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("good")
	})

	app.Post("/update-fps", func(c *fiber.Ctx) error {
		w := &FiberResponseWriter{ctx: c}
		r := convertFiberToHTTP(c)
		
		game_state, err := server_state.GetSessionGameState(r)
		if err != nil {
			return c.Status(500).SendString("Error in new-screen: " + err.Error())
		}

		target_fps_str := c.Query("value")
		target_fps, err := strconv.ParseInt(target_fps_str, 10, 64)
		if err != nil {
			return c.Status(400).SendString("Invalid fps value")
		}

		game_state.SetTargetFPS(int(target_fps))

		game_state.Mut.Lock()
		err = server_state.Templates.ExecuteTemplate(w, "templates/screen-frame.tmpl.html", game_state)
		game_state.Mut.Unlock()
		if err != nil {
			return c.Status(500).SendString("Error running screen-frame template")
		}
		
		c.Set("Content-Type", "text/html")
		return c.Send(w.buf.Bytes())
	})

	app.Get("/get-screen", func(c *fiber.Ctx) error {
		w := &FiberResponseWriter{ctx: c}
		r := convertFiberToHTTP(c)
		
		
		err := server_state.PlayerRequestedFrame(w, r)
		if err != nil {
			return c.Status(500).SendString("Error running get-screen: " + err.Error())
		}

		c.Set("Content-Type", "text/html")
		return c.Send(w.buf.Bytes())
	})

	app.Get("/get-dead-screen", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendString(`<div></div>`) // Return empty div - dead screen is now handled in UI overlay
	})

	app.Get("/empty", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.Status(200).SendString("")
	})

	app.Put("/jump-player", func(c *fiber.Ctx) error {
		w := &FiberResponseWriter{ctx: c}
		r := convertFiberToHTTP(c)
		
		err := server_state.PlayerJumped(w, r)
		if err != nil {
			return c.Status(500).SendString("Error jumping: " + err.Error())
		}


		c.Set("Content-Type", "text/html")
		return c.Send(w.buf.Bytes())
	})

	app.Get("/get-stats", func(c *fiber.Ctx) error {
		w := &FiberResponseWriter{ctx: c}
		r := convertFiberToHTTP(c)

		game_state, err := server_state.GetSessionGameState(r)
		if err != nil {
			return c.Status(500).SendString("Error in jump-player: " + err.Error())
		}

		game_state.Mut.Lock()
		err = server_state.Templates.ExecuteTemplate(w, "templates/stats.tmpl.html", game_state)
		game_state.Mut.Unlock()

		if err != nil {
			return c.Status(500).SendString("Error running template in get-stats: " + err.Error())
		}

		c.Set("Content-Type", "text/html")
		return c.Send(w.buf.Bytes())
	})

	app.Post("/restart-game", func(c *fiber.Ctx) error {
		w := &FiberResponseWriter{ctx: c}
		r := convertFiberToHTTP(c)
		
		// Re-initialize the game for this player
		err := server_state.PlayerEntered(w, r)
		if err != nil {
			return c.Status(500).SendString("Error restarting game: " + err.Error())
		}

		c.Set("Content-Type", "text/html")
		return c.Send(w.buf.Bytes())
	})

	app.Get("/get-ui-overlay", func(c *fiber.Ctx) error {
		w := &FiberResponseWriter{ctx: c}
		r := convertFiberToHTTP(c)
		
		game_state, err := server_state.GetSessionGameState(r)
		if err != nil {
			return c.Status(500).SendString("Error in get-ui-overlay: " + err.Error())
		}

		game_state.Mut.Lock()
		err = server_state.Templates.ExecuteTemplate(w, "templates/ui-overlay.tmpl.html", game_state)
		game_state.Mut.Unlock()

		if err != nil {
			return c.Status(500).SendString("Error running template in get-ui-overlay: " + err.Error())
		}

		c.Set("Content-Type", "text/html")
		return c.Send(w.buf.Bytes())
	})

	app.Post("/set-username", func(c *fiber.Ctx) error {
		username := c.FormValue("username")
		if username == "" {
			// Generate guest name if empty
			username = generateGuestName()
		}

		// Set username cookie
		cookie := &fiber.Cookie{
			Name:     "username",
			Value:    username,
			Path:     "/",
			MaxAge:   86400 * 30, // 30 days
			HTTPOnly: false,      // Allow JS access for display
			Secure:   false,      // Set to true in production with HTTPS
		}
		c.Cookie(cookie)

		// Return the polling section
		c.Set("Content-Type", "text/html")
		pollingHTML := `
		<div id="polling-section">
			<!-- Polling elements will be added after first jump -->
		</div>
		<div id="name-update" 
			 hx-get="/get-ui-overlay"
			 hx-target="#ui-content"
			 hx-swap="innerHTML"
			 hx-trigger="load"
			 style="display: none;">
		</div>`

		return c.SendString(pollingHTML)
	})

	app.Get("/get-name-input", func(c *fiber.Ctx) error {
		// Get current username from cookie
		currentUsername := c.Cookies("username", "")

		c.Set("Content-Type", "text/html")
		nameInputHTML := `
		<h1>Welcome, <span id="player-name-display">` + currentUsername + `</span>!</h1>
		<div id="name-input-section">
			<p style="margin-bottom: 15px;">Choose your name:</p>
			<form id="name-form" style="display: flex; align-items: center; justify-content: center; gap: 10px; flex-wrap: wrap;">
				<input type="text" id="player-name-input" name="username" class="name-input" placeholder="Enter name" maxlength="20" value="` + currentUsername + `">
				<button type="submit" 
						hx-post="/set-username"
						hx-include="#name-form"
						hx-target="#polling-container"
						hx-swap="innerHTML"
						class="save-name-btn">Save</button>
			</form>
			<div style="margin-top: 15px;">
				<button hx-post="/set-username"
						hx-target="#polling-container"
						hx-swap="innerHTML"
						class="guest-name-btn">Play as Guest</button>
			</div>
		</div>
		<div id="game-instructions" style="display: none;">
			<p>Press Space to start and to jump</p>
			<button hx-get="/get-name-input" 
					hx-target="#instructions-card" 
					hx-swap="innerHTML"
					class="change-name-link" style="margin-top: 10px;">Change Name</button>
		</div>`

		return c.SendString(nameInputHTML)
	})

	app.Get("/start-polling", func(c *fiber.Ctx) error {
		r := convertFiberToHTTP(c)
		
		// Get poll rate from game state
		pollRate := "16ms" // Default poll rate
		game_state, err := server_state.GetSessionGameState(r)
		if err == nil {
			game_state.Mut.Lock()
			pollRate = game_state.PollRate
			game_state.Mut.Unlock()
		}

		c.Set("Content-Type", "text/html")
		pollingHTML := `
		<span id="screen-polling"
			hx-trigger="every ` + pollRate + `"
			hx-get="/get-screen"
			hx-swap="innerHTML"
			hx-target="#screen"
			style="display: none;"
		></span>
		<span id="ui-polling"
			hx-trigger="every 500ms"
			hx-get="/get-ui-overlay"
			hx-swap="innerHTML"
			hx-target="#ui-content"
			style="display: none;"
		></span>
		<span
			hx-trigger="every 1s"
			hx-get="/get-stats"
			hx-swap="innerHTML"
			hx-target="#stats"
			style="display: none;"
		></span>`

		return c.SendString(pollingHTML)
	})

	app.Get("/stop-screen-polling", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		// Return empty spans to stop both screen and UI polling
		stoppedPollingHTML := `
		<span id="screen-polling" style="display: none;"></span>
		<span id="ui-polling" style="display: none;"></span>`
		return c.SendString(stoppedPollingHTML)
	})

	app.Get("/trigger-stop-polling", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		c.Set("HX-Trigger", "stop-screen-polling")
		return c.SendString("")
	})


	app.Get("/get-leaderboard", func(c *fiber.Ctx) error {
		w := &FiberResponseWriter{ctx: c}
		c.Set("Content-Type", "text/html")
		
		// Debug: Print all query parameters
		log.Printf("=== LEADERBOARD DEBUG ===")
		log.Printf("Full URL: %s", c.OriginalURL())
		log.Printf("Query string: %s", c.Request().URI().QueryString())
		
		tab := c.Query("tab", "")
		log.Printf("Tab param value: '%s' (len: %d)", tab, len(tab))
		
		if tab == "" {
			tab = "top-scores"
			log.Printf("Tab was empty, defaulting to: %s", tab)
		} else {
			log.Printf("Using tab from query param: %s", tab)
		}
		
		log.Printf("Final ActiveTab value: %s", tab)
		log.Printf("=== END DEBUG ===)")
		
		// Get top 10 scores
		topScores, err := server_state.Dbq.GetTopScores(server_state.Ctx, 10)
		if err != nil {
			log.Printf("Error getting top scores: %v", err)
			return c.SendString("<p>Error loading leaderboard</p>")
		}
		
		// Get recent 5 scores  
		recentScores, err := server_state.Dbq.GetRecentScores(server_state.Ctx, 5)
		if err != nil {
			log.Printf("Error getting recent scores: %v", err)
			return c.SendString("<p>Error loading leaderboard</p>")
		}
		
		leaderboardData := struct {
			TopScores    []db.GetTopScoresRow
			RecentScores []db.GetRecentScoresRow
			ActiveTab    string
		}{
			TopScores:    topScores,
			RecentScores: recentScores,
			ActiveTab:    tab,
		}
		
		log.Printf("Template data - ActiveTab: '%s', TopScores count: %d, RecentScores count: %d", 
			leaderboardData.ActiveTab, len(leaderboardData.TopScores), len(leaderboardData.RecentScores))
		
		err = server_state.Templates.ExecuteTemplate(w, "templates/leaderboard.tmpl.html", leaderboardData)
		if err != nil {
			log.Printf("Error rendering leaderboard template: %v", err)
			return c.SendString("<p>Error rendering leaderboard</p>")
		}
		
		return c.Send(w.buf.Bytes())
	})

	// Add Prometheus metrics endpoint using Fiber's adaptor
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	
	log.Printf("Server starting on port 3200")
	err = app.Listen(":3200")

	if err != nil {
		log.Printf("Error starting server: %v", err)
	}

}
