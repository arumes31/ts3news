package idely

import "ts3news/internal/clientquery"

// SnapshotClientQuery reads the current clients from a connected ClientQuery
// session into the detector's Client form. Idle is left zero here: ClientQuery
// does not expose remote clients' idle time, so the caller fills Idle from an
// ActivityTracker fed by voice-activity events.
func SnapshotClientQuery(c *clientquery.Client) ([]Client, error) {
	infos, err := c.ClientListBasic()
	if err != nil {
		return nil, err
	}
	out := make([]Client, 0, len(infos))
	for _, ci := range infos {
		out = append(out, Client{
			CLID:     ci.CLID,
			CID:      ci.CID,
			UID:      ci.UID,
			Nickname: ci.Nickname,
			Type:     ci.Type,
		})
	}
	return out, nil
}
