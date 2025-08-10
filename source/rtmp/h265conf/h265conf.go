// Package h265conf contains a H265 configuration parser.
package h265conf

import (
	"fmt"
)

// Conf is a RTMP H265 configuration.
type Conf struct {
	VPS []byte
	SPS []byte
	PPS []byte
}

// Unmarshal decodes a Conf from bytes.
func (c *Conf) Unmarshal(buf []byte) error {
	if len(buf) < 23 {
		return fmt.Errorf("invalid H265 configuration size")
	}

	// HEVC Decoder Configuration Record format
	// Skip configurationVersion (1), profile (4), profileCompatibility (4), 
	// tierFlag (1), levelIdc (6), reserved (6), chromaFormat (2), bitDepthLuma (3), bitDepthChroma (3)
	pos := 22

	numOfArrays := int(buf[pos])
	pos++

	for i := 0; i < numOfArrays; i++ {
		if pos+3 > len(buf) {
			return fmt.Errorf("invalid H265 configuration: incomplete array header")
		}

		naluType := buf[pos] & 0x3F
		pos++

		numNalus := int(uint16(buf[pos])<<8 | uint16(buf[pos+1]))
		pos += 2

		for j := 0; j < numNalus; j++ {
			if pos+2 > len(buf) {
				return fmt.Errorf("invalid H265 configuration: incomplete NALU length")
			}

			naluLen := int(uint16(buf[pos])<<8 | uint16(buf[pos+1]))
			pos += 2

			if pos+naluLen > len(buf) {
				return fmt.Errorf("invalid H265 configuration: incomplete NALU data")
			}

			naluData := buf[pos : pos+naluLen]
			pos += naluLen

			switch naluType {
			case 32: // VPS
				if c.VPS == nil {
					c.VPS = make([]byte, len(naluData))
					copy(c.VPS, naluData)
				}
			case 33: // SPS
				if c.SPS == nil {
					c.SPS = make([]byte, len(naluData))
					copy(c.SPS, naluData)
				}
			case 34: // PPS
				if c.PPS == nil {
					c.PPS = make([]byte, len(naluData))
					copy(c.PPS, naluData)
				}
			}
		}
	}

	if c.VPS == nil || c.SPS == nil || c.PPS == nil {
		return fmt.Errorf("H265 configuration missing required parameter sets")
	}

	return nil
}

// Marshal encodes a Conf into bytes.
func (c Conf) Marshal() ([]byte, error) {
	vpsLen := len(c.VPS)
	spsLen := len(c.SPS)
	ppsLen := len(c.PPS)

	// Calculate buffer size: header(23) + 3 arrays with headers and data
	bufSize := 23 + 3 + 3*5 + vpsLen + spsLen + ppsLen
	buf := make([]byte, bufSize)

	// HEVC Decoder Configuration Record
	buf[0] = 1 // configurationVersion
	
	// Copy profile info from SPS (simplified)
	if len(c.SPS) >= 4 {
		buf[1] = c.SPS[1] // general_profile_space + general_tier_flag + general_profile_idc
		buf[2] = c.SPS[2] // general_profile_compatibility_flags[0]
		buf[3] = c.SPS[3] // general_profile_compatibility_flags[1]
		buf[4] = c.SPS[4] // general_profile_compatibility_flags[2]
	}
	
	// Set other fields to reasonable defaults
	buf[12] = 0xF0 // lengthSizeMinusOne = 3, reserved
	buf[22] = 3    // numOfArrays

	pos := 23

	// VPS array
	buf[pos] = 0x20 | 32 // array_completeness + reserved + NAL_unit_type (VPS=32)
	pos++
	buf[pos] = 0 // numNalus high byte
	buf[pos+1] = 1 // numNalus low byte
	pos += 2
	buf[pos] = byte(vpsLen >> 8)
	buf[pos+1] = byte(vpsLen)
	pos += 2
	copy(buf[pos:], c.VPS)
	pos += vpsLen

	// SPS array
	buf[pos] = 0x20 | 33 // array_completeness + reserved + NAL_unit_type (SPS=33)
	pos++
	buf[pos] = 0 // numNalus high byte
	buf[pos+1] = 1 // numNalus low byte
	pos += 2
	buf[pos] = byte(spsLen >> 8)
	buf[pos+1] = byte(spsLen)
	pos += 2
	copy(buf[pos:], c.SPS)
	pos += spsLen

	// PPS array
	buf[pos] = 0x20 | 34 // array_completeness + reserved + NAL_unit_type (PPS=34)
	pos++
	buf[pos] = 0 // numNalus high byte
	buf[pos+1] = 1 // numNalus low byte
	pos += 2
	buf[pos] = byte(ppsLen >> 8)
	buf[pos+1] = byte(ppsLen)
	pos += 2
	copy(buf[pos:], c.PPS)

	return buf[:pos+ppsLen], nil
}