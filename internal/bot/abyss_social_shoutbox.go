package bot

import (
	"errors"
	"log"

	"github.com/lib/pq"

	"ts3news/internal/clientquery"
)

func (b *Bot) flushAbyssShoutbox(client *clientquery.Client, clients []clientquery.ClientInfo) {
	if b == nil || b.DB == nil || client == nil {
		return
	}
	b.updateAbyssCompetitionPresence(clients)
	// Only senders currently in the bot's own channel can be relayed (the
	// ClientQuery channel message always targets the bot's channel), so
	// select deliverable shouts before applying the batch limit.
	self, err := client.WhoAmIInfo()
	if err != nil {
		return
	}
	channels := make(map[string]int, len(clients))
	eligible := make([]string, 0, len(clients))
	for _, member := range clients {
		if member.Type == 0 && member.UID != "" && member.CID >= 0 {
			channels[member.UID] = member.CID
			if member.CID == self.CID {
				eligible = append(eligible, member.UID)
			}
		}
	}
	if len(eligible) == 0 {
		return
	}
	rows, err := b.DB.Query(`SELECT m.message_id,m.sender_uid,
		COALESCE(NULLIF(u.nickname,''),'Adventurer'),m.message
		FROM abyss_shoutbox_messages m JOIN users u ON u.client_uid=m.sender_uid
		WHERE m.relayed_at IS NULL AND m.created_at>NOW()-INTERVAL '10 minutes'
		  AND m.sender_uid = ANY($1)
		ORDER BY m.message_id LIMIT 10`, pq.Array(eligible))
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
		// SendChannelMessage refuses delivery unless the bot currently occupies
		// the sender's channel, so a shout can never land in a channel the
		// sender is not in. Undelivered shouts stay pending until they expire.
		if err := client.SendChannelMessage(cid, abyssShoutboxRelayLine(shout.nick, shout.message)); err != nil {
			if errors.Is(err, clientquery.ErrNotInChannel) {
				continue
			}
			log.Printf("Abyss shoutbox relay failed for message %d: %v", shout.id, err)
			continue
		}
		_, _ = b.DB.Exec("UPDATE abyss_shoutbox_messages SET relayed_at=NOW() WHERE message_id=$1 AND relayed_at IS NULL", shout.id)
	}
}

// abyssShoutboxRelayLine formats a relayed shout. Both the nickname and the
// message are player-controlled, and TeamSpeak renders BBCode in channel
// messages, so their markup is neutralized before the bot posts them under
// its own identity.
func abyssShoutboxRelayLine(nick, message string) string {
	return "[Abyss] " + sanitizeBBCode(nick) + ": " + sanitizeBBCode(message)
}
