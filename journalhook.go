package journalhook

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/coreos/go-systemd/journal"
	logrus "github.com/sirupsen/logrus"
)

type JournalHook struct {
	SortEntries bool
}

var (
	severityMap = map[logrus.Level]journal.Priority{
		logrus.DebugLevel: journal.PriDebug,
		logrus.InfoLevel:  journal.PriInfo,
		logrus.WarnLevel:  journal.PriWarning,
		logrus.ErrorLevel: journal.PriErr,
		logrus.FatalLevel: journal.PriCrit,
		logrus.PanicLevel: journal.PriEmerg,
	}
)

func stringifyOp(r rune) rune {
	// Journal wants uppercase strings. See `validVarName`
	// https://github.com/coreos/go-systemd/blob/ff118ad0f8d9cf99903d3391ca3a295671022cee/journal/journal.go#L137-L147
	switch {
	case r >= 'A' && r <= 'Z':
		return r
	case r >= '0' && r <= '9':
		return r
	case r == '_':
		return r
	case r >= 'a' && r <= 'z':
		return r - 32
	default:
		return rune('_')
	}
}

func stringifyKey(key string) string {
	key = strings.Map(stringifyOp, key)
	if strings.HasPrefix(key, "_") {
		key = strings.TrimPrefix(key, "_")
	}
	return key
}

func stringifyEntries(data map[string]interface{}, sortEntries bool) map[string]string {
	entries := make(map[string]string)
	if !sortEntries {
		for k, v := range data {
			key, value := stringifyEntry(k, v)
			entries[key] = value
		}
		return entries
	}
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		key, value := stringifyEntry(k, data[k])
		entries[key] = value
	}
	return entries
}

// Journal wants strings but logrus takes anything.
func stringifyEntry(k string, v interface{}) (string, string) {
	return stringifyKey(k), fmt.Sprint(v)
}

func (hook *JournalHook) Fire(entry *logrus.Entry) error {
	return journal.Send(entry.Message, severityMap[entry.Level], stringifyEntries(entry.Data, hook.SortEntries))
}

// `Levels()` returns a slice of `Levels` the hook is fired for.
func (hook *JournalHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

// Adds the Journal hook if journal is enabled
// Sets log output to ioutil.Discard so stdout isn't captured.
func Enable() {
	enable(false)
}

// Adds the Journal hook if journal is enabled
// Sets log output to ioutil.Discard so stdout isn't captured.
// Sort entries before writing to journal
func EnableSortEntries() {
	enable(true)
}

func enable(sortEntries bool) {
	if !journal.Enabled() {
		logrus.Warning("Journal not available but user requests we log to it. Ignoring")
	} else {
		logrus.AddHook(&JournalHook{SortEntries: sortEntries})
		logrus.SetOutput(ioutil.Discard)
	}
}
