// mautrix-whatsapp - A Matrix-WhatsApp puppeting bridge.
// Copyright (C) 2019 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package database

import (
	"strings"

	"maunium.net/go/mautrix-whatsapp/types"
)

type PortalKey struct {
	JID      types.WhatsAppID
	Receiver types.WhatsAppID
}

func GroupPortalKey(jid types.WhatsAppID) PortalKey {
	return PortalKey{
		JID:      jid,
		Receiver: jid,
	}
}

func NewPortalKey(jid, receiver types.WhatsAppID) PortalKey {
	if strings.HasSuffix(jid, "@g.us") {
		receiver = jid
	}
	return PortalKey{
		JID:      jid,
		Receiver: receiver,
	}
}

func (key PortalKey) String() string {
	if key.Receiver == key.JID {
		return key.JID
	}
	return key.JID + "-" + key.Receiver
}

type Portal struct {
	Key  PortalKey
	MXID types.MatrixRoomID

	Name      string
	Topic     string
	Avatar    string
	AvatarURL string

	LastMessage *Message
	UserIDs map[types.MatrixUserID]bool
}
