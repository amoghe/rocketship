package host

import (
	"fmt"
	"strings"
)

var (
	defaultGroups = map[string]uint32{
		// name      : GID
		"root":     0,
		"daemon":   1,
		"bin":      2,
		"sys":      3,
		"adm":      4, // TODO: needed by pkgs like rsyslogd
		"tty":      5,
		"disk":     6,
		"proxy":    13, // TODO: delete me?
		"kmem":     15,
		"sudo":     27,
		"www-data": 33,

		"shadow":  42,
		"utmp":    43,
		"plugdev": 46,
		"staff":   50,

		"libuuid":    101,
		"crontab":    102, // TODO: delete me?
		"syslog":     103,
		"fuse":       104,
		"messagebus": 105,
		"ssh":        108,
		"netdev":     110,

		// Groups needed by aerodrome services
		"radio":       1001,
		"crashcorder": 1002,

		// TODO: bespoke

		"nogroup": 65534,
	}
)

func GroupFileEntry(name string, id uint32, users []User) string {
	var usersList []string
	for _, u := range users {
		usersList = append(usersList, u.Name)
	}

	userNames := strings.Join(usersList, ",")

	return fmt.Sprintf("%s:%s:%d:%s",
		name,
		"x",
		id,
		userNames,
	)
}
