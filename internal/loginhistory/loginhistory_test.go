package loginhistory

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/walles/ftop/internal/assert"
	"github.com/walles/ftop/internal/log"
)

func usersAt(lastOutput string, now, testTime time.Time) []string {
	userSet := make(map[string]struct{})
	for _, line := range strings.Split(lastOutput, "\n") {
		user := userAt(line, testTime, now)
		if user != nil {
			userSet[*user] = struct{}{}
		}
	}

	returnValue := make([]string, 0, len(userSet))
	for user := range userSet {
		returnValue = append(returnValue, user)
	}
	slices.Sort(returnValue)

	return returnValue
}

func TestUserAtReturnsNilForNoise(t *testing.T) {
	now := time.Date(2016, time.April, 7, 12, 8, 0, 0, time.Local)
	timestamp := now

	assert.Equal(t, userAt("", timestamp, now) == nil, true)
	assert.Equal(t, userAt("wtmp begins Thu Oct  1 22:54 ", timestamp, now) == nil, true)
	assert.Equal(t, userAt("reboot   system boot  4.2.0-30-generic Thu Mar  3 11:19 - 13:38 (6+02:18)", timestamp, now) == nil, true)
	assert.Equal(t, userAt("shutdown  ~                         Fri Oct 23 06:49 ", timestamp, now) == nil, true)
}

func TestGetUsersAtRange(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	lastLine := "johan     ttys000                   Thu Mar 31 14:39 - 11:08  (20:29)"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 31, 14, 38, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 31, 14, 39, 0, 0, time.Local)), []string{"johan"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 31, 17, 46, 0, 0, time.Local)), []string{"johan"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.April, 1, 11, 8, 0, 0, time.Local)), []string{"johan"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.April, 1, 11, 9, 0, 0, time.Local)), []string{})
}

func TestGetUsersAtRangeRaspbian(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	lastLine := "sk-tv                                  Thu Mar 31 14:39 - down   (20:29)"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 31, 14, 38, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 31, 14, 39, 0, 0, time.Local)), []string{"sk-tv"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 31, 17, 46, 0, 0, time.Local)), []string{"sk-tv"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.April, 1, 11, 8, 0, 0, time.Local)), []string{"sk-tv"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.April, 1, 11, 9, 0, 0, time.Local)), []string{})
}

func TestGetUsersAtStillLoggedIn(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	lastLine := "johan     ttys000                   Sun Apr  3 11:54   still logged in"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.April, 3, 11, 53, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.April, 3, 11, 54, 0, 0, time.Local)), []string{"johan"})
	assert.SlicesEqual(t, usersAt(lastLine, now, now), []string{"johan"})
}

func TestGetUsersAtRemote(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	lastLine := "root     pts/1        10.1.6.120       Tue Jan 28 05:59   still logged in"

	assert.SlicesEqual(t, usersAt(lastLine, now, now), []string{"root from 10.1.6.120"})
}

func TestGetUsersAtLocalLinux(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	lastLine := "johan    pts/2        :0               Wed Mar  9 13:25 - 13:38  (00:12)"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 9, 13, 26, 0, 0, time.Local)), []string{"johan from :0"})
}

func TestGetUsersAtRemoteMoshPidAddress(t *testing.T) {
	now := time.Date(2016, time.December, 6, 9, 21, 0, 0, time.Local)
	lastLine := "norbert  pts/3        mosh [29846]     Wed Oct 24 15:33 - 15:34  (00:01)"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.October, 24, 15, 34, 0, 0, time.Local)), []string{"norbert from mosh"})
}

func TestGetUsersAtUntilCrash(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	lastLine := "johan     ttys001                   Thu Nov 26 19:55 - crash (27+07:11)"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2015, time.November, 26, 19, 54, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2015, time.November, 26, 19, 55, 0, 0, time.Local)), []string{"johan"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2015, time.December, 10, 19, 53, 0, 0, time.Local)), []string{"johan"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2015, time.December, 26, 19, 55, 0, 0, time.Local)), []string{})
}

func TestGetUsersAtUntilShutdownOsx(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	lastLine := "_mbsetupuser  console                   Mon Jan 18 20:31 - shutdown (34+01:29)"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.January, 18, 20, 30, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.January, 18, 20, 31, 0, 0, time.Local)), []string{"_mbsetupuser"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.February, 18, 20, 30, 0, 0, time.Local)), []string{"_mbsetupuser"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.February, 28, 20, 30, 0, 0, time.Local)), []string{})
}

func TestGetUsersAtUntilShutdownLinux(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	lastLine := "johan    :0           :0               Sat Mar 26 22:04 - down   (00:08)"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 26, 22, 3, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 26, 22, 4, 0, 0, time.Local)), []string{"johan from :0"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 26, 22, 9, 0, 0, time.Local)), []string{"johan from :0"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.March, 26, 22, 15, 0, 0, time.Local)), []string{})
}

func TestGetUsersAtLoginScreenAddress(t *testing.T) {
	now := time.Date(2025, time.December, 3, 14, 23, 2, 0, time.Local)
	lastLine := "gaffatejp  seat0        login screen     Sat Aug 24 11:22 - 13:33  (02:10)"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2025, time.August, 24, 11, 21, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2025, time.August, 24, 11, 22, 0, 0, time.Local)), []string{"gaffatejp from login screen"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2025, time.August, 24, 13, 32, 0, 0, time.Local)), []string{"gaffatejp from login screen"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2025, time.August, 24, 13, 33, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2025, time.August, 24, 13, 34, 0, 0, time.Local)), []string{})
}

func TestGetUsersAtLoginScreenCrashAndDown(t *testing.T) {
	now := time.Date(2025, time.December, 3, 14, 23, 2, 0, time.Local)
	crashLine := "gaffatejp  seat0        login screen     Sat Aug 24 15:55 - crash  (09:21)"
	downLine := "gaffatejp  seat0        login screen     Sat Aug 24 13:45 - down   (01:47)"

	assert.SlicesEqual(t, usersAt(crashLine, now, time.Date(2025, time.August, 24, 15, 55, 0, 0, time.Local)), []string{"gaffatejp from login screen"})
	assert.SlicesEqual(t, usersAt(crashLine, now, time.Date(2025, time.August, 25, 1, 16, 0, 0, time.Local)), []string{"gaffatejp from login screen"})
	assert.SlicesEqual(t, usersAt(crashLine, now, time.Date(2025, time.August, 25, 1, 17, 0, 0, time.Local)), []string{})

	assert.SlicesEqual(t, usersAt(downLine, now, time.Date(2025, time.August, 24, 13, 45, 0, 0, time.Local)), []string{"gaffatejp from login screen"})
	assert.SlicesEqual(t, usersAt(downLine, now, time.Date(2025, time.August, 24, 15, 32, 0, 0, time.Local)), []string{"gaffatejp from login screen"})
	assert.SlicesEqual(t, usersAt(downLine, now, time.Date(2025, time.August, 24, 15, 33, 0, 0, time.Local)), []string{})
}

func TestGetUsersAtTreatsGoneNoLogoutAsStillLoggedIn(t *testing.T) {
	now := time.Date(2016, time.April, 7, 12, 8, 0, 0, time.Local)
	lastLine := "johan    pts/3        :0               Mon Apr  4 23:10    gone - no logout"

	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.April, 4, 23, 9, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.April, 4, 23, 10, 0, 0, time.Local)), []string{"johan from :0"})
	assert.SlicesEqual(t, usersAt(lastLine, now, time.Date(2016, time.April, 7, 12, 8, 0, 0, time.Local)), []string{"johan from :0"})
}

func TestGetUsersAtMultiple(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	lastOutput := strings.Join([]string{
		"johan1     ttys000                   Thu Mar 31 14:39 - 11:08  (20:29)",
		"johan2     ttys000                   Thu Mar 31 14:39 - 11:08  (20:29)",
	}, "\n")

	assert.SlicesEqual(t, usersAt(lastOutput, now, time.Date(2016, time.March, 31, 14, 38, 0, 0, time.Local)), []string{})
	assert.SlicesEqual(t, usersAt(lastOutput, now, time.Date(2016, time.March, 31, 14, 39, 0, 0, time.Local)), []string{"johan1", "johan2"})
	assert.SlicesEqual(t, usersAt(lastOutput, now, time.Date(2016, time.March, 31, 17, 46, 0, 0, time.Local)), []string{"johan1", "johan2"})
	assert.SlicesEqual(t, usersAt(lastOutput, now, time.Date(2016, time.April, 1, 11, 8, 0, 0, time.Local)), []string{"johan1", "johan2"})
	assert.SlicesEqual(t, usersAt(lastOutput, now, time.Date(2016, time.April, 1, 11, 9, 0, 0, time.Local)), []string{})
}

func TestGetUsersAtIgnoresPseudoUsersAndNoise(t *testing.T) {
	now := time.Date(2016, time.April, 7, 12, 8, 0, 0, time.Local)
	lastOutput := "reboot   system boot  4.2.0-30-generic Thu Mar  3 11:19 - 13:38 (6+02:18)\nshutdown  ~                         Fri Oct 23 06:49 \nwtmp begins Thu Oct  1 22:54 "

	assert.SlicesEqual(t, usersAt(lastOutput, now, now), []string{})
}

func TestUserAtLogsUnexpectedLastOutput(t *testing.T) {
	now := time.Date(2016, time.April, 7, 12, 8, 0, 0, time.Local)
	unexpected := "glasskiosk unexpected last line marker"

	assert.Equal(t, userAt(unexpected, now, now) == nil, true)
	assert.Equal(t, strings.Contains(log.String(false), unexpected), true)
}

func TestGetUsersAtSmoke(t *testing.T) {
	_, err := GetUsersAt(time.Now())
	assert.Equal(t, err, nil)
}

func TestToTimestampHandlesLeapYearsAndPreviousYear(t *testing.T) {
	now := time.Date(2016, time.April, 3, 12, 8, 0, 0, time.Local)
	timestamp, err := toTimestamp("Thu Mar  5 11:19", now)
	assert.Equal(t, err, nil)
	assert.Equal(t, timestamp, time.Date(2016, time.March, 5, 11, 19, 0, 0, time.Local))

	timestamp, err = toTimestamp("Mon Feb 29 13:19", now)
	assert.Equal(t, err, nil)
	assert.Equal(t, timestamp, time.Date(2016, time.February, 29, 13, 19, 0, 0, time.Local))

	now = time.Date(2017, time.January, 3, 12, 8, 0, 0, time.Local)
	timestamp, err = toTimestamp("Mon Feb 29 13:19", now)
	assert.Equal(t, err, nil)
	assert.Equal(t, timestamp, time.Date(2016, time.February, 29, 13, 19, 0, 0, time.Local))
}

func TestToDuration(t *testing.T) {
	duration, err := toDuration("01:29")
	assert.Equal(t, err, nil)
	assert.Equal(t, duration, time.Hour+29*time.Minute)

	duration, err = toDuration("4+01:29")
	assert.Equal(t, err, nil)
	assert.Equal(t, duration, 4*24*time.Hour+time.Hour+29*time.Minute)

	duration, err = toDuration("34+01:29")
	assert.Equal(t, err, nil)
	assert.Equal(t, duration, 34*24*time.Hour+time.Hour+29*time.Minute)
}
