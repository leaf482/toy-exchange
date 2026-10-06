package replay

import (
	"bytes"
	"fmt"
)

// Replay feeds a JSONL capture that is already in memory.
// Blank lines and lines that start with # are skipped.
// A decode or sequence error names the 1-based line.
func (s *Session) Replay(data []byte, onUpdate func()) error {
	lineNo := 0
	for len(data) > 0 {
		i := bytes.IndexByte(data, '\n')
		var line []byte
		if i < 0 {
			line = data
			data = nil
		} else {
			line = data[:i]
			data = data[i+1:]
		}
		lineNo++
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		msg, err := Decode(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
		if err := s.Feed(msg, onUpdate); err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
	}
	return nil
}
