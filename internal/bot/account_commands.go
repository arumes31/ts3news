package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"ts3news/internal/clientquery"
)

func (s *Supervisor) handleAccountRecoveryCommand(c *clientquery.Client, event string, params map[string]string) {
	if event != "notifytextmessage" || params["targetmode"] != "1" || s.bot.Cfg.WebBaseURL == "" {
		return
	}
	clid, err := strconv.Atoi(params["invokerid"])
	if err != nil || clid <= 0 || params["invokeruid"] == "" {
		return
	}
	// Re-resolve the live client ID before sending a secret: IDs are reused after
	// disconnects. Never choose the account from message text or display names.
	uid, err := c.ClientUID(clid)
	if err != nil || uid != params["invokeruid"] {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	token, err := s.bot.issueAccountRecovery(ctx, uid)
	message := "Password setup is unavailable. Wait a minute and try !password again while connected with the same identity."
	if err == nil {
		message = fmt.Sprintf("Set or reset your portal password: %s/account/recover?token=%s\nThis private link expires in 10 minutes and works once. Never share it.", strings.TrimRight(s.bot.Cfg.WebBaseURL, "/"), token)
	}
	// Check again after the database operation to reduce the client-ID reuse window.
	currentUID, lookupErr := c.ClientUID(clid)
	if lookupErr != nil || currentUID != uid {
		return
	}
	if err := c.SendPrivateMessage(clid, message); err != nil {
		log.Print("web account recovery private message could not be delivered")
	}
}
