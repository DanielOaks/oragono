// Copyright (c) 2012-2014 Jeremy Latt
// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package irc

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/goshuirc/irc-go/ircfmt"
	"github.com/oragono/oragono/irc/history"
	"github.com/oragono/oragono/irc/sno"
	"github.com/oragono/oragono/irc/utils"
)

var (
	restrictedNicknames = []string{
		"=scene=",  // used for rp commands
		"HistServ", // used to play back JOIN, PART, etc. to legacy clients
	}

	restrictedCasefoldedNicks = make(map[string]bool)
	restrictedSkeletons       = make(map[string]bool)
)

// returns whether the change succeeded or failed
func performNickChange(server *Server, client *Client, target *Client, session *Session, nickname string, rb *ResponseBuffer) bool {
	currentNick := client.Nick()
	details := target.Details()
	hadNick := details.nick != "*"
	origNickMask := details.nickMask

	assignedNickname, err := client.server.clients.SetNick(target, session, nickname)
	if err == errNoop {
		return true
	} else if err == errNicknameInUse {
		rb.Add(nil, server.name, ERR_NICKNAMEINUSE, currentNick, utils.SafeErrorParam(nickname), client.t("Nickname is already in use"))
	} else if err == errNicknameReserved {
		rb.Add(nil, server.name, ERR_NICKNAMEINUSE, currentNick, utils.SafeErrorParam(nickname), client.t("Nickname is reserved by a different account"))
	} else if err == errNicknameInvalid {
		rb.Add(nil, server.name, ERR_ERRONEUSNICKNAME, currentNick, utils.SafeErrorParam(nickname), client.t("Erroneous nickname"))
	} else if err == errNickAccountMismatch {
		// this used to use ERR_NICKNAMEINUSE, but it displayed poorly in some clients;
		// ERR_UNKNOWNERROR at least has a better chance of displaying our error text
		rb.Add(nil, server.name, ERR_UNKNOWNERROR, currentNick, "NICK", client.t(err.Error()))
	} else if err == errNickMissing {
		rb.Add(nil, server.name, ERR_NONICKNAMEGIVEN, currentNick, client.t("No nickname given"))
	} else if err != nil {
		rb.Add(nil, server.name, ERR_UNKNOWNERROR, currentNick, "NICK", fmt.Sprintf(client.t("Could not set or change nickname: %s"), err.Error()))
	}
	if err != nil {
		return false
	}

	message := utils.MakeMessage("")
	histItem := history.Item{
		Type:        history.Nick,
		Nick:        origNickMask,
		AccountName: details.accountName,
		Message:     message,
	}
	histItem.Params[0] = assignedNickname

	client.server.logger.Debug("nick", fmt.Sprintf("%s changed nickname to %s [%s]", origNickMask, assignedNickname, client.NickCasefolded()))
	if hadNick {
		if client == target {
			target.server.snomasks.Send(sno.LocalNicks, fmt.Sprintf(ircfmt.Unescape("$%s$r changed nickname to %s"), details.nick, assignedNickname))
		} else {
			target.server.snomasks.Send(sno.LocalNicks, fmt.Sprintf(ircfmt.Unescape("Operator %s changed nickname of $%s$r to %s"), client.Nick(), details.nick, assignedNickname))
		}
		target.server.whoWas.Append(details.WhoWas)
		rb.AddFromClient(message.Time, message.Msgid, origNickMask, details.accountName, nil, "NICK", assignedNickname)
		for session := range target.Friends() {
			if session != rb.session {
				session.sendFromClientInternal(false, message.Time, message.Msgid, origNickMask, details.accountName, nil, "NICK", assignedNickname)
			}
		}
	}

	for _, channel := range client.Channels() {
		channel.AddHistoryItem(histItem)
	}

	if target.Registered() {
		client.server.monitorManager.AlertAbout(target, true)
		target.nickTimer.Touch(rb)
	} // else: these will be deferred to the end of registration (see #572)
	return true
}

func (server *Server) RandomlyRename(client *Client) {
	format := server.Config().Accounts.NickReservation.GuestFormat
	buf := make([]byte, 8)
	rand.Read(buf)
	nick := strings.Replace(format, "*", utils.B32Encoder.EncodeToString(buf), -1)
	sessions := client.Sessions()
	if len(sessions) == 0 {
		return
	}
	// XXX arbitrarily pick the first session to receive error messages;
	// all other sessions receive a `NICK` line same as a friend would
	rb := NewResponseBuffer(sessions[0])
	performNickChange(server, client, client, nil, nick, rb)
	rb.Send(false)
	// technically performNickChange can fail to change the nick,
	// but if they're still delinquent, the timer will get them later
}

// if force-nick-equals-account is set, account name and nickname must be equal,
// so we need to re-NICK automatically on every login event (IDENTIFY,
// VERIFY, and a REGISTER that auto-verifies). if we can't get the nick
// then we log them out (they will be able to reattach with SASL)
func fixupNickEqualsAccount(client *Client, rb *ResponseBuffer, config *Config) (success bool) {
	if !config.Accounts.NickReservation.ForceNickEqualsAccount {
		return true
	}
	if !client.registered {
		return true
	}
	// don't need to supply a nickname, SetNick will use the account name
	if !performNickChange(client.server, client, client, rb.session, "", rb) {
		client.server.accounts.Logout(client)
		nsNotice(rb, client.t("A client is already using that account; try logging out and logging back in with SASL"))
		return false
	}
	return true
}
