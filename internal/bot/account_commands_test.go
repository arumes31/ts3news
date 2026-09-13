package bot

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/clientquery"
	"ts3news/internal/config"
)

func TestAccountRecoveryCommandRejectsUnverifiedChannels(t *testing.T) {
	s := &Supervisor{bot: &Bot{Cfg: &config.Config{WebBaseURL: "https://example.test"}}}
	for _, tc := range []struct{ name, event, mode, uid string }{
		{"channel message", "notifytextmessage", "2", "player"},
		{"server message", "notifytextmessage", "3", "player"},
		{"poke", "notifypoke", "1", "player"},
		{"missing identity", "notifytextmessage", "1", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Nil ClientQuery and DB ensure rejection occurs before any external action.
			s.handleAccountRecoveryCommand(nil, tc.event, map[string]string{"targetmode": tc.mode, "invokeruid": tc.uid, "invokerid": "7"})
		})
	}
}

func TestAccountRecoveryCommandVerifiesPrivateRecipient(t *testing.T) {
	for _, tc := range []struct {
		name, firstUID, secondUID string
		update, send              bool
	}{
		{"verified sender", "player", "player", true, true},
		{"mismatched sender", "impostor", "impostor", false, false},
		{"client id reused", "player", "new-player", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if tc.update {
				mock.ExpectExec("UPDATE users SET web_recovery_hash").WithArgs(sqlmock.AnyArg(), "player").WillReturnResult(sqlmock.NewResult(0, 1))
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			messages := make(chan []string, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					messages <- nil
					return
				}
				defer conn.Close()
				fmt.Fprintln(conn, "error id=0 msg=ok")
				scan := bufio.NewScanner(conn)
				var sent []string
				lookup := 0
				for scan.Scan() {
					command := scan.Text()
					if strings.HasPrefix(command, "clientvariable ") {
						uid := tc.firstUID
						if lookup > 0 {
							uid = tc.secondUID
						}
						lookup++
						fmt.Fprintln(conn, "client_unique_identifier="+uid)
					} else if strings.HasPrefix(command, "sendtextmessage ") {
						sent = append(sent, command)
					}
					fmt.Fprintln(conn, "error id=0 msg=ok")
				}
				messages <- sent
			}()
			client, err := clientquery.Dial(listener.Addr().String(), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			s := &Supervisor{bot: &Bot{DB: db, Cfg: &config.Config{WebBaseURL: "https://example.test"}}}
			s.handleAccountRecoveryCommand(client, "notifytextmessage", map[string]string{"targetmode": "1", "invokeruid": "player", "invokerid": "7"})
			client.Close()
			sent := <-messages
			if tc.send {
				if len(sent) != 1 || !strings.Contains(sent[0], "targetmode=1 target=7") || !strings.Contains(sent[0], "account\\/recover?token=") {
					t.Fatal("verified recovery was not sent privately to its owner")
				}
			} else if len(sent) != 0 {
				t.Fatal("sent recovery credential to an unverified recipient")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
