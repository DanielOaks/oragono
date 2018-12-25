// Copyright (c) 2017 Shivaram Lingamneni <slingamn@cs.stanford.edu>
// released under the MIT license

package irc

import (
	"github.com/unendingPattern/oragono/irc/isupport"
	"github.com/unendingPattern/oragono/irc/modes"
)

func (server *Server) Config() *Config {
	server.configurableStateMutex.RLock()
	defer server.configurableStateMutex.RUnlock()
	return server.config
}

func (server *Server) ISupport() *isupport.List {
	server.configurableStateMutex.RLock()
	defer server.configurableStateMutex.RUnlock()
	return server.isupport
}

func (server *Server) Limits() Limits {
	return server.Config().Limits
}

func (server *Server) Password() []byte {
	return server.Config().Server.passwordBytes
}

func (server *Server) RecoverFromErrors() bool {
	return *server.Config().Debug.RecoverFromErrors
}

func (server *Server) ProxyAllowedFrom() []string {
	return server.Config().Server.ProxyAllowedFrom
}

func (server *Server) WebIRCConfig() []webircConfig {
	return server.Config().Server.WebIRC
}

func (server *Server) DefaultChannelModes() modes.Modes {
	return server.Config().Channels.defaultModes
}

func (server *Server) ChannelRegistrationEnabled() bool {
	return server.Config().Channels.Registration.Enabled
}

func (server *Server) AccountConfig() *AccountConfig {
	return &server.Config().Accounts
}

func (server *Server) FakelagConfig() *FakelagConfig {
	return &server.Config().Fakelag
}

func (server *Server) GetOperator(name string) (oper *Oper) {
	name, err := CasefoldName(name)
	if err != nil {
		return
	}
	server.configurableStateMutex.RLock()
	defer server.configurableStateMutex.RUnlock()
	return server.config.operators[name]
}

func (client *Client) Nick() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.nick
}

func (client *Client) NickMaskString() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.nickMaskString
}

func (client *Client) NickCasefolded() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.nickCasefolded
}

func (client *Client) NickMaskCasefolded() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.nickMaskCasefolded
}

func (client *Client) Username() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.username
}

func (client *Client) Hostname() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.hostname
}

func (client *Client) Realname() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.realname
}

func (client *Client) Oper() *Oper {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.oper
}

func (client *Client) SetOper(oper *Oper) {
	client.stateMutex.Lock()
	defer client.stateMutex.Unlock()
	client.oper = oper
}

func (client *Client) Registered() bool {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.registered
}

func (client *Client) Destroyed() bool {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.isDestroyed
}

func (client *Client) Account() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.account
}

func (client *Client) AccountName() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	if client.accountName == "" {
		return "*"
	}
	return client.accountName
}

func (client *Client) SetAccountName(account string) (changed bool) {
	var casefoldedAccount string
	var err error
	if account != "" {
		if casefoldedAccount, err = CasefoldName(account); err != nil {
			return
		}
	}

	client.stateMutex.Lock()
	defer client.stateMutex.Unlock()
	changed = client.account != casefoldedAccount
	client.account = casefoldedAccount
	client.accountName = account
	return
}

func (client *Client) Authorized() bool {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.authorized
}

func (client *Client) SetAuthorized(authorized bool) {
	client.stateMutex.Lock()
	defer client.stateMutex.Unlock()
	client.authorized = authorized
}

func (client *Client) PreregNick() string {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	return client.preregNick
}

func (client *Client) SetPreregNick(preregNick string) {
	client.stateMutex.Lock()
	defer client.stateMutex.Unlock()
	client.preregNick = preregNick
}

func (client *Client) HasMode(mode modes.Mode) bool {
	// client.flags has its own synch
	return client.flags.HasMode(mode)
}

func (client *Client) SetMode(mode modes.Mode, on bool) bool {
	return client.flags.SetMode(mode, on)
}

func (client *Client) Channels() (result []*Channel) {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()
	length := len(client.channels)
	result = make([]*Channel, length)
	i := 0
	for channel := range client.channels {
		result[i] = channel
		i++
	}
	return
}

func (client *Client) WhoWas() (result WhoWas) {
	client.stateMutex.RLock()
	defer client.stateMutex.RUnlock()

	result.nicknameCasefolded = client.nickCasefolded
	result.nickname = client.nick
	result.username = client.username
	result.hostname = client.hostname
	result.realname = client.realname

	return
}

func (channel *Channel) Name() string {
	channel.stateMutex.RLock()
	defer channel.stateMutex.RUnlock()
	return channel.name
}

func (channel *Channel) setName(name string) {
	channel.stateMutex.Lock()
	defer channel.stateMutex.Unlock()
	channel.name = name
}

func (channel *Channel) NameCasefolded() string {
	channel.stateMutex.RLock()
	defer channel.stateMutex.RUnlock()
	return channel.nameCasefolded
}

func (channel *Channel) setNameCasefolded(nameCasefolded string) {
	channel.stateMutex.Lock()
	defer channel.stateMutex.Unlock()
	channel.nameCasefolded = nameCasefolded
}

func (channel *Channel) Members() (result []*Client) {
	channel.stateMutex.RLock()
	defer channel.stateMutex.RUnlock()
	return channel.membersCache
}

func (channel *Channel) UserLimit() uint64 {
	channel.stateMutex.RLock()
	defer channel.stateMutex.RUnlock()
	return channel.userLimit
}

func (channel *Channel) setUserLimit(limit uint64) {
	channel.stateMutex.Lock()
	channel.userLimit = limit
	channel.stateMutex.Unlock()
}

func (channel *Channel) Key() string {
	channel.stateMutex.RLock()
	defer channel.stateMutex.RUnlock()
	return channel.key
}

func (channel *Channel) setKey(key string) {
	channel.stateMutex.Lock()
	defer channel.stateMutex.Unlock()
	channel.key = key
}

func (channel *Channel) HighLights() string {
	channel.stateMutex.RLock()
	defer channel.stateMutex.RUnlock()
	return channel.highlights
}

func (channel *Channel) setHighLights(highlights string) {
	channel.stateMutex.Lock()
	defer channel.stateMutex.Unlock()
	channel.highlights = highlights
}

func (channel *Channel) Founder() string {
	channel.stateMutex.RLock()
	defer channel.stateMutex.RUnlock()
	return channel.registeredFounder
}
