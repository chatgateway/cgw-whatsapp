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

package main

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"io/ioutil"
	"os"
	"time"

	"maunium.net/go/mautrix-whatsapp/database"
	"maunium.net/go/mautrix-whatsapp/types"
)

func load(path string, into interface{}) error {
	if file, err := os.OpenFile(path, os.O_RDONLY, 0600); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	} else if reader, err := gzip.NewReader(file); err != nil {
		_ = file.Close()
		return err
	} else if err = gob.NewDecoder(reader).Decode(into); err != nil {
		_ = reader.Close()
		_ = file.Close()
		return err
	} else if err = reader.Close(); err != nil {
		_ = file.Close()
		return err
	} else {
		return file.Close()
	}
}

func save(path string, data interface{}) error {
	if file, err := os.OpenFile(path+".tmp", os.O_WRONLY|os.O_CREATE, 0600); err != nil {
		return err
	} else if writer := gzip.NewWriter(file); false {
		return errors.New("false is true")
	} else if err = gob.NewEncoder(writer).Encode(data); err != nil {
		_ = writer.Close()
		_ = file.Close()
		return err
	} else if err = writer.Close(); err != nil {
		_ = file.Close()
		return err
	} else if err = file.Close(); err != nil {
		return err
	} else {
		return os.Rename(path+".tmp", path)
	}
}

func (bridge *Bridge) LoadNextBatch() error {
	dat, err := ioutil.ReadFile(bridge.Config.AppService.Database.Sync)
	if err != nil {
		bridge.AS.Sync.NextBatch = ""
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	bridge.AS.Sync.NextBatch = string(dat)
	return nil
}

func (bridge *Bridge) SaveNextBatch() error {
	return ioutil.WriteFile(bridge.Config.AppService.Database.Sync,  []byte(bridge.AS.Sync.NextBatch), 0600)
}

func (bridge *Bridge) SaveLoop() {
	bridge.Log.Debugln("Starting db save loop")
	tick := time.NewTicker(30 * time.Second)
	prevNextBatch := bridge.AS.Sync.NextBatch
	for range tick.C {
		select {
		case <-tick.C:
			if bridge.usersChanged {
				err := bridge.SaveUsers()
				if err != nil {
					bridge.Log.Warnln("Failed to save users:", err)
				}
				bridge.usersChanged = false
			}
			if bridge.portalsChanged {
				err := bridge.SavePortals()
				if err != nil {
					bridge.Log.Warnln("Failed to save portals:", err)
				}
				bridge.portalsChanged = false
			}
			if bridge.puppetsChanged {
				err := bridge.SavePuppets()
				if err != nil {
					bridge.Log.Warnln("Failed to save puppets:", err)
				}
				bridge.puppetsChanged = false
			}
			if prevNextBatch != bridge.AS.Sync.NextBatch {
				err := bridge.SaveNextBatch()
				if err != nil {
					bridge.Log.Warnln("Failed to write next batch token:", err)
				}
				prevNextBatch = bridge.AS.Sync.NextBatch
			}
		case <-bridge.stopSaveLoop:
			return
		}
	}
}

func (bridge *Bridge) SaveMessage(message *database.Message) {
	// TODO
}

func (bridge *Bridge) DeleteMessage(message *database.Message) {
	// TODO
}

func (bridge *Bridge) GetMessageByMXID(mxid types.MatrixEventID) *database.Message {
	// TODO
	return nil
}

func (bridge *Bridge) GetMessageByJID(portal database.PortalKey, jid types.WhatsAppMessageID) *database.Message {
	// TODO
	return nil
}

func (bridge *Bridge) LoadUsers() error {
	var data []*database.User
	if err := load(bridge.Config.AppService.Database.Users, &data); err != nil {
		return err
	}
	bridge.usersLock.Lock()
	for _, dbUser := range data {
		user, ok := bridge.usersByMXID[dbUser.MXID]
		if ok {
			user.User = dbUser
			continue
		}
		user = bridge.NewUser(dbUser)
		bridge.usersByMXID[user.MXID] = user
		if len(user.JID) > 0 {
			bridge.usersByJID[user.JID] = user
		}
	}
	bridge.usersLock.Unlock()
	return nil
}

func (bridge *Bridge) LoadPortals() error {
	var data []*database.Portal
	if err := load(bridge.Config.AppService.Database.Portals, &data); err != nil {
		return err
	}
	bridge.portalsLock.Lock()
	for _, dbPortal := range data {
		portal, ok := bridge.portalsByJID[dbPortal.Key]
		if ok {
			portal.Portal = dbPortal
			continue
		}
		portal = bridge.NewPortal(dbPortal)
		bridge.portalsByJID[portal.Key] = portal
		if len(portal.MXID) > 0 {
			bridge.portalsByMXID[portal.MXID] = portal
		}
	}
	bridge.portalsLock.Unlock()
	return nil
}

func (bridge *Bridge) LoadPuppets() error {
	var data []*database.Puppet
	if err := load(bridge.Config.AppService.Database.Puppets, &data); err != nil {
		return err
	}
	bridge.puppetsLock.Lock()
	for _, dbPuppet := range data {
		puppet, ok := bridge.puppets[dbPuppet.JID]
		if ok {
			puppet.Puppet = dbPuppet
			continue
		}
		puppet = bridge.NewPuppet(dbPuppet)
		bridge.puppets[puppet.JID] = puppet
		if len(puppet.CustomMXID) > 0 {
			bridge.puppetsByCustomMXID[puppet.CustomMXID] = puppet
		}
	}
	bridge.puppetsLock.Unlock()
	return nil
}

func (bridge *Bridge) SaveUsers() error {
	bridge.usersLock.Lock()
	bridge.Log.Debugln("Saving users...")
	data := make([]*database.User, len(bridge.usersByMXID))
	i := 0
	for _, user := range bridge.usersByMXID {
		data[i] = user.User
		i++
	}
	bridge.usersLock.Unlock()
	if err := save(bridge.Config.AppService.Database.Users, &data); err != nil {
		return err
	}
	bridge.Log.Debugln("Users saved")
	return nil
}

func (bridge *Bridge) SavePortals() error {
	bridge.portalsLock.Lock()
	bridge.Log.Debugln("Saving portals...")
	data := make([]*database.Portal, len(bridge.portalsByJID))
	i := 0
	for _, portal := range bridge.portalsByJID {
		data[i] = portal.Portal
		i++
	}
	bridge.portalsLock.Unlock()
	if err := save(bridge.Config.AppService.Database.Portals, &data); err != nil {
		return err
	}
	bridge.Log.Debugln("Portals saved")
	return nil
}

func (bridge *Bridge) SavePuppets() error {
	bridge.puppetsLock.Lock()
	bridge.Log.Debugln("Saving puppets...")
	data := make([]*database.Puppet, len(bridge.puppets))
	i := 0
	for _, puppet := range bridge.puppets {
		data[i] = puppet.Puppet
		i++
	}
	bridge.puppetsLock.Unlock()
	if err := save(bridge.Config.AppService.Database.Puppets, &data); err != nil {
		return err
	}
	bridge.Log.Debugln("Puppets saved")
	return nil
}
