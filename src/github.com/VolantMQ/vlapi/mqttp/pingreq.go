// Copyright (c) 2014 The VolantMQ Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mqttp

// PingReq The PINGREQ Packet is sent from a Client to the Server. It can be used to:
// 1. Indicate to the Server that the Client is alive in the absence of any other
//    Control Packets being sent from the Client to the Server.
// 2. Request that the Server responds to confirm that it is alive.
// 3. Exercise the network to indicate that the Network Connection is active.
type PingReq struct {
	header
}

var _ IFace = (*PingReq)(nil)

func newPingReq() *PingReq {
	return &PingReq{}
}

// NewPingReq creates a new PINGREQ packet
func NewPingReq(v ProtocolVersion) *PingReq {
	p := newPingReq()
	p.init(PINGREQ, v, p.size, p.encodeMessage, p.decodeMessage)
	return p
}

// decode message
func (msg *PingReq) decodeMessage(src []byte) (int, error) {
	return 0, nil
}

func (msg *PingReq) encodeMessage(dst []byte) (int, error) {
	return 0, nil
}

// Len of message
func (msg *PingReq) size() int {
	return 0
}
