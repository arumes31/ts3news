package idely

import (
	"time"
	"ts3news/internal/clientquery"
)

// SnapshotClientQuery reads the current clients from a connected ClientQuery
// session into the detector's Client form. It queries client_idle_time directly
// from TeamSpeak client variables.
func SnapshotClientQuery(c *clientquery.Client) ([]Client, error) {
	infos, err := c.ClientListBasic()
	if err != nil {
		return nil, err
	}
	out := make([]Client, 0, len(infos))
	for _, ci := range infos {
		var idle time.Duration
		var hasIdle bool
		if ci.Type == 0 {
			if ms, err := c.ClientIdleTime(ci.CLID); err == nil {
				idle = time.Duration(ms) * time.Millisecond
				hasIdle = true
			}
		}
		out = append(out, Client{
			CLID:     ci.CLID,
			CID:      ci.CID,
			UID:      ci.UID,
			Nickname: ci.Nickname,
			Type:     ci.Type,
			Idle:     idle,
			HasIdle:  hasIdle,
		})
	}
	return out, nil
}
