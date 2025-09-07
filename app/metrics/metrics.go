package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Player count - current active players
	ActivePlayers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "flappybird_active_players",
		Help: "Current number of active players (alive sessions)",
	})

	// Total connected sessions
	ConnectedSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "flappybird_connected_sessions",
		Help: "Current number of connected sessions (including dead players)",
	})

	// Frame rates per session
	SessionFPS = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "flappybird_session_fps",
		Help: "Current FPS for each active session",
	}, []string{"session_id"})

	// Session scores
	SessionScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "flappybird_session_score",
		Help: "Current score for each session",
	}, []string{"session_id"})

	// Session target FPS
	SessionTargetFPS = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "flappybird_session_target_fps",
		Help: "Target FPS setting for each session",
	}, []string{"session_id"})

	// Total frames served (counter that increments)
	TotalFramesServed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "flappybird_frames_served_total",
		Help: "Total number of frames served since server start",
	})

	// Frames served in last sample period (gauge that resets every 5 seconds)
	FramesServedLastSample = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "flappybird_frames_served_last_sample",
		Help: "Number of frames served in the last 5-second sample period",
	})

	// Average FPS across all active sessions
	AverageFPS = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "flappybird_average_fps",
		Help: "Average FPS across all active sessions",
	})
)

// CleanupSessionMetrics removes metrics for a disconnected session
func CleanupSessionMetrics(sessionID string) {
	SessionFPS.DeleteLabelValues(sessionID)
	SessionScore.DeleteLabelValues(sessionID)
	SessionTargetFPS.DeleteLabelValues(sessionID)
}