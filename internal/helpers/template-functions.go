package helpers

import (
	"net/http"
	"time"
)

func addTemplateFunctions(r *http.Request) {
	views.AddGlobal("humanDate", func(t time.Time) string {
		return HumanDate(t)
	})

	views.AddGlobal("dateFromLayout", func(t time.Time, l string) string {
		return FormatDateWithLayout(t, l)
	})

	views.AddGlobal("dateAfterYearOne", func(t time.Time) bool {
		return DateAfterY1(t)
	})

	views.AddGlobal("isAdmin", func() bool {
		return IsAdmin(r)
	})

}

// HumanDate formats a time in YYYY-MM-DD format
func HumanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02-01-2006")
}

// FormatDateWithLayout formats a time with provided (go compliant) format string, and returns it as a string
func FormatDateWithLayout(t time.Time, f string) string {
	return t.Format(f)
}

// DateAfterY1 is used to verify that a date is after the year 1 (since go hates nulls)
func DateAfterY1(t time.Time) bool {
	yearOne := time.Date(0001, 11, 17, 20, 34, 58, 651387237, time.UTC)
	return t.After(yearOne)
}

// IsAdmin is used in templateData to check access level of authenticated user
// and renders certain admin level functions on some pages if true
func IsAdmin(r *http.Request) bool {
	access_level := app.Session.Get(r.Context(), "access_level")
	return access_level == 10
}

// IsAuthenticated is used in templateData to check user is authenticated
// before rendering pages in private routes.
func IsAuthenticated(r *http.Request) bool {
	exists := app.Session.Exists(r.Context(), "username")
	return exists
}

func UserProjects(r *http.Request) map[int]string {
	projects := app.Session.Get(r.Context(), "projects").(map[int]string)
	return projects
}
