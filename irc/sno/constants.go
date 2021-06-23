// Copyright (c) 2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

// Package sno holds Server Notice masks for easy reference.
package sno

// Mask is a type of server notice mask.
type Mask rune

type Masks []Mask

// Notice mask types
const (
	LocalAnnouncements Mask = 'a'
	LocalConnects      Mask = 'c'
	LocalDisconnects   Mask = 'd'
	LocalChannels      Mask = 'j'
	LocalKills         Mask = 'k'
	LocalNicks         Mask = 'n'
	LocalOpers         Mask = 'o'
	LocalQuits         Mask = 'q'
	Stats              Mask = 't'
	LocalAccounts      Mask = 'u'
	LocalVhosts        Mask = 'v'
	LocalXline         Mask = 'x'
)

var (
	// NoticeMaskNames has readable names for our snomask types.
	NoticeMaskNames = map[Mask]string{
		LocalAnnouncements: "ANNOUNCEMENT",
		LocalConnects:      "CONNECT",
		LocalDisconnects:   "DISCONNECT",
		LocalChannels:      "CHANNEL",
		LocalKills:         "KILL",
		LocalNicks:         "NICK",
		LocalOpers:         "OPER",
		LocalQuits:         "QUIT",
		Stats:              "STATS",
		LocalAccounts:      "ACCOUNT",
		LocalXline:         "XLINE",
		LocalVhosts:        "VHOST",
	}

	// ValidMasks contains the snomasks that we support.
	ValidMasks = []Mask{
		LocalAnnouncements,
		LocalConnects,
		LocalDisconnects,
		LocalChannels,
		LocalKills,
		LocalNicks,
		LocalOpers,
		LocalQuits,
		Stats,
		LocalAccounts,
		LocalVhosts,
		LocalXline,
	}
)
