package parser

import "fmt"

// Section
type Section struct {
	file *File

	pos position

	ws0        string
	name       string
	ws1        string
	subsection *SubSection
	ws2        string
	comment    *Comment

	keys []string
}

// New Section
func NewSection(pos position, ws0, name, ws1 string, subsection *SubSection, ws2 string, comment *Comment) *Section {
	keys := make([]string, 0)

	return &Section{
		pos: pos,

		ws0:        ws0,
		name:       name,
		ws1:        ws1,
		subsection: subsection,
		ws2:        ws2,
		comment:    comment,

		keys: keys,
	}
}

// Stringer
func (s Section) String() string {
	subsection := ""
	if s.subsection != nil {
		subsection = s.subsection.String()
	}

	comment := ""
	if s.comment != nil {
		comment = s.comment.String()
	}

	return fmt.Sprintf("[%s%s%s%s]%s%s", s.ws0, s.name, s.ws1, subsection, s.ws2, comment)
}

// Get name
func (s *Section) Name() string {
	return s.name
}

// Retrieve defined keys
func (s *Section) RawKeys() []string {
	return s.keys
}

// Retrieve subsection name
func (s *Section) SubSectionName() string {
	return s.subsection.name
}

// Retrieve defined keys
// Keys are passed through KeyManipFunc
func (s *Section) Keys() []string {
	keys := make([]string, len(s.keys))
	for i, k := range s.keys {
		keys[i] = KeyManipFunc(k)
	}

	return keys
}

// determine insert location, which is the first blank line after a non-blank
func (s *Section) getInsertLocation(idx int) int {
	for i := idx; i >= 0; i-- {
		if s.file.lines[i].item != nil {
			return i + 1
		}
	}

	return -1
}

// Returns the KeyValuePair and its line position, or nil and the position the
// key should be inserted
func (s *Section) getKey(key string) (*KeyValuePair, int) {
	// loop over lines and find the key
	lastSectionName := ""
	for lastIdx, l := range s.file.lines {
		switch l.item.(type) {
		case *Section:
			// if we are entering a new section, and not found, break
			if lastSectionName == s.name {
				return nil, s.getInsertLocation(lastIdx - 1)
			}

			sect, _ := l.item.(*Section)
			lastSectionName = sect.name

		case *KeyValuePair:
			kvp, _ := l.item.(*KeyValuePair)
			//fmt.Printf(">>> compare: %s//%s :: %s//%s\n", lastSectionName, s.name, kvp.key, key)
			if lastSectionName == s.name && KeyCompFunc(kvp.key, key) {
				return kvp, lastIdx
			}
		}
	}

	// if we get here, then must be last section of file
	return nil, s.getInsertLocation(len(s.file.lines) - 1)
}

// Retrieve the raw value for a key
func (s *Section) GetRaw(key string) string {
	k, _ := s.getKey(key)
	if k != nil {
		return k.value
	}

	return ""
}

// Retrieve the value for a key
// The returned value is passed through ValueManipFunc
func (s *Section) Get(key string) string {
	return ValueManipFunc(s.GetRaw(key))
}

// Set raw key with raw value
// If key already present, then it is overwritten
// If key doesn't exist, then it is added to the end of the section
func (s *Section) SetKeyValueRaw(key, value string) {
	//fmt.Printf(">>>>>>>>>>>>>>> line count: %d SetKeyValueRaw: %s.%s//%s\n", len(s.file.lines), s.Name(), key, value)
	//fmt.Printf(">>> file: '%s'\n\n\n", s.file)
	k, pos := s.getKey(key)

	// key is present, set value
	if k != nil {
		k.value = value
		return
	}

	//fmt.Printf(">>>>>>>>>>> pos: %d\n", pos)

	// key doesn't exist, create it...

	// grab default whitespace and line ending
	ws := DefaultLeadingKeyWhitespace

	// set no ws if empty section
	if s.name == "" {
		ws = ""
	}

	le := DefaultLineEnding
	if len(s.file.lines) > 0 {
		// take line ending from first line if present
		le = s.file.lines[0].le
	}

	// create the key and line
	k = NewKeyValuePair(position{}, key, "", value, nil)
	line := NewLine(position{}, ws, k, le)

	// insert line into s.file.lines
	if pos < 0 {
		// must be inserting into empty section where there are no keys present
		s.file.lines = append([]*Line{line}, s.file.lines...)
	} else {
		// copy whitespace from previous line if its a kvp
		if _, ok := s.file.lines[pos-1].item.(*KeyValuePair); ok {
			line.ws = s.file.lines[pos-1].ws
		}

		s.file.lines = append(
			s.file.lines[:pos],
			append(
				[]*Line{line},
				s.file.lines[pos:]...,
			)...,
		)
	}

	// add key to s.keys
	s.keys = append(s.keys, k.key)
}

// Set key with value
// If key already present, then it is overwritten
// If key doesn't exist, then it is added to the end of the section
// Passes key through KeyManipFunc and value through ValueManipFunc
func (s *Section) SetKey(key, value string) {
	s.SetKeyValueRaw(KeyManipFunc(key), ValueManipFunc(value))
}

// Remove key from section
func (s *Section) RemoveKey(key string) {
	k, pos := s.getKey(key)
	if k != nil {
		s.file.lines = append(s.file.lines[:pos], s.file.lines[pos+1:]...)

		// find place in s.keys
		idx := 0
		for ; idx < len(s.keys); idx++ {
			if KeyCompFunc(key, s.keys[idx]) {
				break
			}
		}

		// remove from s.keys
		s.keys = append(s.keys[:idx], s.keys[idx+1:]...)
	}
}
