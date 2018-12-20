// Copyright (c) 2012-2014 Jeremy Latt
// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package irc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/goshuirc/irc-go/ircfmt"
	"github.com/unendingPattern/oragono/irc/sno"
)

var (
	restrictedNicknames = map[string]bool{
		"=scene=": true, // used for rp commands
		"--": true, //used for notifications
	}
)

// returns whether the change succeeded or failed
func performNickChange(server *Server, client *Client, target *Client, newnick string, rb *ResponseBuffer) bool {
	nickname := strings.TrimSpace(newnick)
	cfnick, err := CasefoldName(nickname)

	if len(nickname) < 1 {
		rb.Add(nil, server.name, ERR_NONICKNAMEGIVEN, client.nick, client.t("No nickname given"))
		return false
	}

	if err != nil || len(nickname) > server.Limits().NickLen || restrictedNicknames[cfnick] {
		rb.Add(nil, server.name, ERR_ERRONEUSNICKNAME, client.nick, nickname, client.t("Erroneous nickname"))
		return false
	}

	if target.Nick() == nickname {
		return true
	}

	hadNick := target.HasNick()
	origNickMask := target.NickMaskString()
	whowas := client.WhoWas()
	err = client.server.clients.SetNick(target, nickname)
	if err == errNicknameInUse {
		rb.Add(nil, server.name, ERR_NICKNAMEINUSE, client.nick, nickname, client.t("Nickname is already in use"))
		server.forceNick(nickname, client)
		//return false
	} else if err == errNicknameReserved {
		rb.Add(nil, server.name, ERR_NICKNAMEINUSE, client.nick, nickname, client.t("Nickname is reserved by a different account"))
		server.forceNick(nickname, client)
		//return false
	} else if err != nil {
		rb.Add(nil, server.name, ERR_UNKNOWNERROR, client.nick, "NICK", fmt.Sprintf(client.t("Could not set or change nickname: %s"), err.Error()))
		return false
	}

	client.nickTimer.Touch()

	client.server.logger.Debug("nick", fmt.Sprintf("%s changed nickname to %s [%s]", origNickMask, nickname, cfnick))
	if hadNick {
		target.server.snomasks.Send(sno.LocalNicks, fmt.Sprintf(ircfmt.Unescape("$%s$r changed nickname to %s"), whowas.nickname, nickname))
		target.server.whoWas.Append(whowas)
		for friend := range target.Friends() {
			friend.Send(nil, origNickMask, "NICK", nickname)
		}
	}

	if target.Registered() {
		client.server.monitorManager.AlertAbout(target, true)
	}
	// else: Run() will attempt registration immediately after this
	return true
}

func (server *Server) RandomlyRename(client *Client) {
	prefix := server.AccountConfig().NickReservation.RenamePrefix
	if prefix == "" {
		prefix = "Guest-"
	}
	buf := make([]byte, 8)
	rand.Read(buf)
	nick := fmt.Sprintf("%s%s", prefix, hex.EncodeToString(buf))
	rb := NewResponseBuffer(client)
	performNickChange(server, client, client, nick, rb)
	rb.Send()
	// technically performNickChange can fail to change the nick,
	// but if they're still delinquent, the timer will get them later
}

func (server *Server) forceNick(currentNick string, client *Client) {
	if currentNick == "" || currentNick == "*" {
		currentNick = client.preregNick
	}
	buf := make([]byte, 4)
	rand.Read(buf)
	nick := fmt.Sprintf("%s-%s", currentNick, hex.EncodeToString(buf))
	rb := NewResponseBuffer(client)
	performNickChange(server, client, client, nick, rb)
	rb.Send()
	// technically performNickChange can fail to change the nick,
	// but if they're still delinquent, the timer will get them later
}
