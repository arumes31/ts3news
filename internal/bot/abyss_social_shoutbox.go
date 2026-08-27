package bot

import (
	"log"

	"ts3news/internal/clientquery"
)

func (b *Bot) flushAbyssShoutbox(client *clientquery.Client, clients []clientquery.ClientInfo) {
	if b == nil || b.DB == nil || client == nil {
		return
	}
	b.updateAbyssCompetitionPresence(clients)
	channels := make(map[string]int, len(clients))
	for _, member := range clients {
		if member.Type == 0 && member.UID != "" && member.CID >= 0 {
			channels[member.UID] = member.CID
		}
	}
	rows, err := b.DB.Query(`SELECT m.message_id,m.sender_uid,
		COALESCE(NULLIF(u.nickname,''),'Adventurer'),m.message
		FROM abyss_shoutbox_messages m JOIN users u ON u.client_uid=m.sender_uid
		WHERE m.relayed_at IS NULL AND m.created_at>NOW()-INTERVAL '10 minutes'
		ORDER BY m.message_id LIMIT 10`)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()
	type pendingShout struct {
		id      int64
		uid     string
		nick    string
		message string
	}
	var pending []pendingShout
	for rows.Next() {
		var shout pendingShout
		if rows.Scan(&shout.id, &shout.uid, &shout.nick, &shout.message) != nil {
			return
		}
		pending = append(pending, shout)
	}
	for _, shout := range pending {
		cid, online := channels[shout.uid]
		if !online {
			continue
		}
		if err := client.SendChannelMessage(cid, "[Abyss] "+shout.nick+": "+shout.message); err != nil {
			log.Printf("Abyss shoutbox relay failed for message %d: %v", shout.id, err)
			continue
		}
		_, _ = b.DB.Exec("UPDATE abyss_shoutbox_messages SET relayed_at=NOW() WHERE message_id=$1 AND relayed_at IS NULL", shout.id)
	}
}
