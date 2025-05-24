package main

import (
	"fmt"
	"time"
)

func utc() time.Time {
	return time.Now().UTC()
}

func utc_string() string {
	return utc().Format("2006-01-02T15:04:05-07:00")
}

func format_timestamp(t time.Time) string {
	nowTime := fmt.Sprintf("%d", t.UnixNano())
	return nowTime
}

func format_date_from_iso_utc_to_locale(date string) string {
	const layout = "2006-01-02T15:04:05-07:00"
    t, err := time.Parse(layout, date)
	if didFail(err, "could not parse date") {
		return ""
	}
	return t.Local().Format("Monday, 02 January 2006 15:04 MST")
}