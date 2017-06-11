package encoding

import (
	"bytes"
	"encoding/binary"
	"log"
)

type MessagePriority uint8

const maxApdu = 50
const Normal MessagePriority = 0
const Urgent MessagePriority = 1
const Critical MessagePriority = 2
const LifeSafety MessagePriority = 3

type ConfirmedService uint8

const ReadPropertyService ConfirmedService = 12

const NetworkMessageInvalid = 0xFF

const BacnetProtocolVersion = 1
const HopCountDefault = 255

// Network Protocol Data Units
type npdu struct {
	ExpectingReply      bool
	ProtocolVersion     uint8
	NetworkLayerMessage bool
	NetworkMessageType  uint8
	VendorID            uint16
	Priority            MessagePriority
	HopCount            uint8
}

type BacnetAddress struct {
	Net    uint16
	Len    uint8
	MacLen uint8
	Mac    []uint8
	Adr    []uint8
}

// buff, dest, my addr
func EncodePDU(n *npdu, src *BacnetAddress, dest *BacnetAddress) (b []byte, err error) {
	buff := new(bytes.Buffer)

	// write is a helper function to prevent a ton of "if err != nil" lines and
	// also to ensure that Endian is consistent through out the write process
	write := func(p interface{}) {
		if err != nil {
			return
		}
		err = binary.Write(buff, binary.LittleEndian, p)
	}

	// Writes the bacnet address to the buffer
	writeAddr := func(a *BacnetAddress) {
		// Encode destination
		write(a.Net)
		write(a.Len)
		for _, a := range a.Adr {
			write(a)
		}
	}

	write(n.ProtocolVersion)
	// Several portions of information goes into the next bit
	var temp uint8 = 0
	if n.NetworkLayerMessage {
		temp |= 1 << 7
	}
	// Bit 6: Reserved
	if dest.Net > 0 {
		temp |= 1 << 5
	}
	// Bit 4: Reserved
	if src.Net > 0 && src.Len > 0 {
		temp |= 1 << 3
	}

	if n.ExpectingReply {
		temp |= 1 << 2
	}

	temp |= uint8(n.Priority) & 0x03
	write(temp)

	writeAddr(dest)
	writeAddr(src)

	if dest.Net > 0 {
		write(n.HopCount)
	}

	if n.NetworkLayerMessage {
		write(n.NetworkMessageType)
		if n.NetworkMessageType >= 0x80 {
			write(n.VendorID)
		}
	}

	return buff.Bytes(), err
}

func encodeNPDU(expectingReply bool, priority MessagePriority) npdu {
	return npdu{
		ExpectingReply:      expectingReply,
		ProtocolVersion:     BacnetProtocolVersion,
		NetworkLayerMessage: false,
		NetworkMessageType:  NetworkMessageInvalid,
		VendorID:            0,
		Priority:            priority,
		HopCount:            HopCountDefault,
	}
}

const pduTypeConfirmedServiceRequest = 0

/* from clause 20.1.2.4 max-segments-accepted and clause 20.1.2.5 max-APDU-length-accepted
returns the encoded octet */
func encodeMaxSegsMaxApdu(maxSegs int, maxApdu int) uint8 {
	var octet uint8 = 0

	// 6 is chosen since 2^6 is 64 at which point we hit special cases
	var i uint
	for i = 0; i < 6; i++ {
		log.Println(1 << (i + 1))
		if maxSegs < 1<<(i+1) {
			octet = uint8(i << 4)
			break
		}
	}
	if maxSegs == 64 {
		octet = 0x60
	} else if maxSegs > 64 {
		octet = 0x70
	}

	/* max_apdu must be 50 octets minimum */
	if maxApdu <= 50 {
		octet |= 0x00
	} else if maxApdu <= 128 {
		octet |= 0x01
		/*fits in a LonTalk frame */
	} else if maxApdu <= 206 {
		octet |= 0x02
		/*fits in an ARCNET or MS/TP frame */
	} else if maxApdu <= 480 {
		octet |= 0x03
	} else if maxApdu <= 1024 {
		octet |= 0x04
		/* fits in an ISO 8802-3 frame */
	} else if maxApdu <= 1476 {
		octet |= 0x05
	}

	return octet
}