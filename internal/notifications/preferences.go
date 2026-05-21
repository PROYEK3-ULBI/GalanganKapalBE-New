package notifications

import "strings"

// preferenceKeyByCategory maps the `category` value attached to a notification
// to the key used inside users.notification_preferences (JSONB).
//
// The keys MUST match the option keys declared in the frontend Settings page
// (sims-frontend/src/pages/Settings/Settings.jsx — NOTIF_OPTIONS).
//
// Categories not listed here are treated as "always send" (no opt-out).
var preferenceKeyByCategory = map[string]string{
	"material-request":   "prApproval",
	"material_request":   "prApproval",
	"low-stock":          "lowStock",
	"low_stock":          "lowStock",
	"out-of-stock":       "outOfStock",
	"out_of_stock":       "outOfStock",
	"receipt":            "newReceipt",
	"new-receipt":        "newReceipt",
	"tool-calibration":   "toolCalibration",
	"tool_calibration":   "toolCalibration",
	"daily-report":       "dailyReport",
	"daily_report":       "dailyReport",
}

// preferenceKey returns the JSON key for the given category, or "" if the
// category is unmapped (callers should treat empty as "no filter").
func preferenceKey(category string) string {
	c := strings.ToLower(strings.TrimSpace(category))
	if c == "" {
		return ""
	}
	return preferenceKeyByCategory[c]
}
