package loginhistory

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/walles/ftop/internal/log"
	"github.com/walles/ftop/internal/util"
)

var months = map[string]time.Month{
	"Jan": time.January,
	"Feb": time.February,
	"Mar": time.March,
	"Apr": time.April,
	"May": time.May,
	"Jun": time.June,
	"Jul": time.July,
	"Aug": time.August,
	"Sep": time.September,
	"Oct": time.October,
	"Nov": time.November,
	"Dec": time.December,
}

var lastLineRe = regexp.MustCompile(
	`^(\S+)(?: +(.+?))? +(... ... .. ..:..) [- ] [^(]*( *\(([0-9+:]+)\))?$`,
)

var durationRe = regexp.MustCompile(`^(([0-9]+)\+)?([0-9][0-9]):([0-9][0-9])$`)

func GetUsersAt(timestamp time.Time) ([]string, error) {
	users := make(map[string]struct{})
	// util.Exec() strips locale environment variables, but preserves the TZ
	// environment variable. "last" formats timestamps in that TZ, so parse using
	// the target timestamp's location to keep the times comparable.
	now := time.Now().In(timestamp.Location())

	err := util.Exec([]string{"last"}, func(line string) error {
		user := userAt(line, timestamp, now)
		if user != nil {
			users[*user] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sorted := make([]string, 0, len(users))
	for user := range users {
		sorted = append(sorted, user)
	}
	slices.Sort(sorted)

	return sorted, nil
}

func userAt(line string, timestamp, now time.Time) *string {
	if line == "" {
		return nil
	}
	if strings.HasPrefix(line, "wtmp begins") {
		return nil
	}
	if strings.HasPrefix(line, "reboot ") {
		return nil
	}
	if strings.HasPrefix(line, "shutdown ") {
		return nil
	}

	match := lastLineRe.FindStringSubmatch(line)
	if match == nil {
		log.Infof("Unmatched last line: <%s>", line)
		return nil
	}

	username := match[1]
	address := addressFromMetadata(match[2])
	fromString := match[3]
	durationString := match[5]

	if address != "" {
		username += " from " + address
	}

	fromTimestamp, err := toTimestamp(fromString, now)
	if err != nil {
		log.Infof("Problematic last line: <%s>: %v", line, err)
		return nil
	}
	if timestamp.Before(fromTimestamp) {
		return nil
	}

	if durationString == "" {
		return &username
	}

	duration, err := toDuration(durationString)
	if err != nil {
		log.Infof("Problematic last line: <%s>: %v", line, err)
		return nil
	}

	toTimestamp := fromTimestamp.Add(duration)
	if timestamp.After(toTimestamp) {
		return nil
	}

	return &username
}

func addressFromMetadata(metadata string) string {
	if metadata == "" {
		return ""
	}

	fields := strings.Fields(metadata)
	if len(fields) < 2 {
		return ""
	}

	lastField := fields[len(fields)-1]
	if strings.HasPrefix(lastField, "[") && strings.HasSuffix(lastField, "]") {
		fields = fields[:len(fields)-1]
	}
	if len(fields) < 2 {
		return ""
	}

	return strings.Join(fields[1:], " ")
}

func toTimestamp(stringValue string, now time.Time) (time.Time, error) {
	fields := strings.Fields(stringValue)
	if len(fields) != 4 {
		return time.Time{}, fmt.Errorf("unexpected timestamp format <%s>", stringValue)
	}

	month, ok := months[fields[1]]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown month <%s>", fields[1])
	}

	day, err := strconv.Atoi(fields[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day <%s>: %w", fields[2], err)
	}

	parts := strings.Split(fields[3], ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("unexpected time format <%s>", fields[3])
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour <%s>: %w", parts[0], err)
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute <%s>: %w", parts[1], err)
	}

	timestamp, err := buildTimestamp(now.Year(), month, day, hour, minute, now.Location())
	if err == nil && !timestamp.After(now) {
		return timestamp, nil
	}
	if err != nil && (month != time.February || day != 29) {
		return time.Time{}, err
	}

	return buildTimestamp(now.Year()-1, month, day, hour, minute, now.Location())
}

func buildTimestamp(year int, month time.Month, day, hour, minute int, location *time.Location) (time.Time, error) {
	timestamp := time.Date(year, month, day, hour, minute, 0, 0, location)
	if timestamp.Month() != month || timestamp.Day() != day || timestamp.Hour() != hour || timestamp.Minute() != minute {
		return time.Time{}, fmt.Errorf("invalid timestamp %04d-%02d-%02d %02d:%02d", year, month, day, hour, minute)
	}

	return timestamp, nil
}

func toDuration(stringValue string) (time.Duration, error) {
	match := durationRe.FindStringSubmatch(stringValue)
	if match == nil {
		return 0, fmt.Errorf("unexpected duration format <%s>", stringValue)
	}

	days := 0
	if match[2] != "" {
		var err error
		days, err = strconv.Atoi(match[2])
		if err != nil {
			return 0, fmt.Errorf("invalid days <%s>: %w", match[2], err)
		}
	}

	hours, err := strconv.Atoi(match[3])
	if err != nil {
		return 0, fmt.Errorf("invalid hours <%s>: %w", match[3], err)
	}

	minutes, err := strconv.Atoi(match[4])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes <%s>: %w", match[4], err)
	}

	return time.Duration(days)*24*time.Hour + time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
}
